/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package login

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes/docker"
	dockerconfig "github.com/containerd/containerd/remotes/docker/config"
	"github.com/containerd/log"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/dockerutil"
	"github.com/containerd/nerdctl/v2/pkg/errutil"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/term"
)

const unencryptedPasswordWarning = `WARNING: Your password will be stored unencrypted in %s.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store
`

func Login(ctx context.Context, options types.LoginCommandOptions, stdout io.Writer) error {
	// If we cannot even parse the address, bail out
	serverURL, err := dockerutil.Parse(options.ServerAddress)
	if err != nil {
		return fmt.Errorf("failed parsing requested server url %q (%w)", options.ServerAddress, err)
	}

	// Get a credentialStore (does not error on ENOENT).
	// If it errors, it is a hard filesystem error or a JSON parsing error, and login in that context does not make sense, so, just stop.
	credentialsStore, err := dockerutil.New("")
	if err != nil {
		return fmt.Errorf("credentials store is broken for registry %q (%w)", options.ServerAddress, err)
	}

	// Get an authconfig object for that registry, and only check the store if no username nor password was provided
	// If it errored, we'll deal with it later
	authConfig, err := credentialsStore.GetAuthConfigFor(serverURL, options.Username == "" && options.Password == "")
	if err != nil {
		//  else {
		//				log.L.WithError(err).Warnf("cannot get auth config for authConfigHostname=%q (refHostname=%q)",
		//					identifier, serverURL.Host)
		//			}

	}

	// Delete any possible identityToken from there
	authConfig.IdentityToken = ""

	// Get the hosts dirs that exist and that we can read
	hostsDirs := dockerutil.ValidateDirectories(options.GOptions.HostsDir)

	// Prepare host options
	ho := &dockerconfig.HostOptions{
		DefaultScheme: "https",
		Credentials:   credentialsStore.GetCredentialsFunction(ctx, authConfig),
	}

	// Attach the hosts resolution function, dealing with both explicit port and without (if 443)
	ho.HostDir = func(s string) (string, error) {
		for _, hostsDir := range hostsDirs {
			found, err := dockerconfig.HostDirFromRoot(hostsDir)(s)
			if (err != nil && !errdefs.IsNotFound(err)) || (found != "") {
				return found, err
			}
			// Try again without the port if it is standard
			if serverURL.Port() == "443" {
				// no need to reparse from s as s = serverURL.String()
				found, err = dockerconfig.HostDirFromRoot(hostsDir)(serverURL.Hostname())
				if (err != nil && !errdefs.IsNotFound(err)) || (found != "") {
					return found, err
				}
			}
		}
		return "", nil
	}

	// Set to insecure if asked by the user, or if it is localhost and the user did NOT set the flag explicitly (to false)
	if options.GOptions.InsecureRegistry || (docker.IsLocalhost(serverURL.Hostname()) && !options.GOptions.ExplicitInsecureRegistry) {
		log.G(ctx).Warnf("by using insecure registry, we are skipping verifying HTTPS certs and will allow downgrading to plain HTTP for %q", serverURL.Host)
		ho.DefaultTLS = &tls.Config{
			InsecureSkipVerify: true,
		}
		// Shouldn't we set that after the first failure instead?
		ho.DefaultScheme = "http"
	}

	// Get all configured hosts for that server
	regHosts, configHostsErr := dockerconfig.ConfigureHosts(ctx, *ho)(serverURL.Host)
	if configHostsErr != nil {
		return configHostsErr
	}

	if len(regHosts) == 0 {
		return fmt.Errorf("got empty []docker.RegistryHost for %q", serverURL.Host)
	}

	var responseIdentityToken string
	// If we found a username and password from the store, and it did not error, then try to log in with that
	if err == nil && authConfig.Username != "" && authConfig.Password != "" {
		responseIdentityToken, err = loginClientSide(ctx, ho, regHosts, options.GOptions.InsecureRegistry)
	}

	// if we had an error reading from the CredentialStore, or if above failed, or we did not have username / password, ask the user for what's missing and try (again)
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		err = promptUserForAuthentication(authConfig, options.Username, options.Password)
		if err != nil {
			return fmt.Errorf("failed reading credentials %w", err)
		}

		responseIdentityToken, err = loginClientSide(ctx, ho, regHosts, options.GOptions.InsecureRegistry)
		if err != nil {
			return fmt.Errorf("failed logging in with provided credentials %w", err)
		}
	}

	// If we got an identity token back, this is what we are going to store instead of the password
	if responseIdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = responseIdentityToken
	}

	// Display a warning if we're storing the users password (not a token) and credentials store type is file.
	if authConfig.Password != "" {
		filename := credentialsStore.IsStoreFile(serverURL)

		if filename != "" {
			_, err = fmt.Fprintln(stdout, fmt.Sprintf(unencryptedPasswordWarning, filename))
			if err != nil {
				return err
			}
		}
	}

	if err = credentialsStore.Save(authConfig); err != nil {
		return fmt.Errorf("error during login: %w", err)
	}

	_, err = fmt.Fprintln(stdout, "Login Succeeded")

	return err
}

func loginClientSide(ctx context.Context, ho *dockerconfig.HostOptions, regHosts []docker.RegistryHost, insecure bool) (string, error) {
	fetchedRefreshTokens := make(map[string]string) // key: req.URL.Host
	// onFetchRefreshToken is called when tryLoginWithRegHost calls rh.Authorizer.Authorize()
	onFetchRefreshToken := func(ctx context.Context, s string, req *http.Request) {
		fetchedRefreshTokens[req.URL.Host] = s
	}
	ho.AuthorizerOpts = append(ho.AuthorizerOpts, docker.WithFetchRefreshToken(onFetchRefreshToken))

	var err error
	for i, rh := range regHosts {
		err = tryLoginWithRegHost(ctx, rh)
		if err != nil && insecure && (errutil.IsErrHTTPResponseToHTTPSClient(err) || errutil.IsErrConnectionRefused(err)) {
			rh.Scheme = "http"
			err = tryLoginWithRegHost(ctx, rh)
		}
		identityToken := fetchedRefreshTokens[rh.Host] // can be empty
		if err == nil {
			return identityToken, nil
		}
		log.G(ctx).WithError(err).WithField("i", i).Error("failed to call tryLoginWithRegHost")
	}
	return "", err
}

func tryLoginWithRegHost(ctx context.Context, rh docker.RegistryHost) error {
	if rh.Authorizer == nil {
		return errors.New("got nil Authorizer")
	}
	if rh.Path == "/v2" {
		// If the path is using /v2 endpoint but lacks trailing slash add it
		// https://docs.docker.com/registry/spec/api/#detail. Acts as a workaround
		// for containerd issue https://github.com/containerd/containerd/blob/2986d5b077feb8252d5d2060277a9c98ff8e009b/remotes/docker/config/hosts.go#L110
		rh.Path = "/v2/"
	}
	u := url.URL{
		Scheme: rh.Scheme,
		Host:   rh.Host,
		Path:   rh.Path,
	}
	var ress []*http.Response
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		for k, v := range rh.Header.Clone() {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
		if err = rh.Authorizer.Authorize(ctx, req); err != nil {
			return fmt.Errorf("failed to call rh.Authorizer.Authorize: %w", err)
		}
		res, err := ctxhttp.Do(ctx, rh.Client, req)
		if err != nil {
			return fmt.Errorf("failed to call rh.Client.Do: %w", err)
		}
		ress = append(ress, res)
		if res.StatusCode == 401 {
			if err = rh.Authorizer.AddResponses(ctx, ress); err != nil && !errdefs.IsNotImplemented(err) {
				return fmt.Errorf("failed to call rh.Authorizer.AddResponses: %w", err)
			}
			continue
		}
		if res.StatusCode/100 != 2 {
			return fmt.Errorf("unexpected status code %d", res.StatusCode)
		}

		return nil
	}

	return errors.New("too many 401 (probably)")
}

func promptUserForAuthentication(authConfig *dockerutil.AuthConfig, username, password string) error {
	// If the provided username is empty, use the one we know of
	if username = strings.TrimSpace(username); username == "" {
		username = authConfig.Username
	}
	// If the one from the store is empty as well, read it
	if username == "" {
		fmt.Print("Enter Username: ")
		usr, err := readUsername()
		if err != nil {
			return err
		}
		username = usr
	}
	// If it still is empty, that is an error
	if username == "" {
		return fmt.Errorf("error: Username is Required")
	}

	// If password was NOT passed along, ask for it
	if password == "" {
		fmt.Print("Enter Password: ")
		pwd, err := readPassword()
		fmt.Println()
		if err != nil {
			return err
		}
		password = pwd
	}
	// If nothing was provided, error out
	if password == "" {
		return fmt.Errorf("error: Password is Required")
	}

	// Attach that to the auth object
	authConfig.Username = username
	authConfig.Password = password

	return nil
}

func readUsername() (string, error) {
	var fd *os.File
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fd = os.Stdin
	} else {
		return "", fmt.Errorf("stdin is not a terminal (Hint: use `nerdctl login --username=USERNAME --password-stdin`)")
	}

	reader := bufio.NewReader(fd)
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading username: %w", err)
	}
	username = strings.TrimSpace(username)

	return username, nil
}
