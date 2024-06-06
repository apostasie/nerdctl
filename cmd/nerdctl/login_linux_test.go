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

// https://docs.docker.com/reference/cli/dockerd/#insecure-registries
// Local registries, whose IP address falls in the 127.0.0.0/8 range, are automatically marked as insecure as of Docker 1.3.2.
// It isn't recommended to rely on this, as it may change in the future.
// "--insecure" means that either the certificates are untrusted, or that the protocol is plain http

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/testca"
	"github.com/containerd/nerdctl/v2/pkg/testutil/testregistry"
)

func createTempDir(t *testing.T, mode os.FileMode) string {
	tmpDir, err := os.MkdirTemp(t.TempDir(), "docker-config")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(tmpDir, mode)
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func setDockerConfigLocation(t *testing.T, path string) {
	err := os.Setenv("DOCKER_CONFIG", path)
	if err != nil {
		t.Fatal(err)
	}
}

func getDockerConfigLocation() string {
	return os.Getenv("DOCKER_CONFIG")
}

func TestLoginLocalhostNoAuthStd(t *testing.T) {
	base := testutil.NewBase(t)
	// Group setup
	localhostHTTPOnlyNoAuthStd := testregistry.NewRegistry(base, nil, 80, &testregistry.NoAuth{})
	old := getDockerConfigLocation()
	// This guarantees docker credential file isolation between tests
	setup := func() {
		tmpDir := createTempDir(t, 0700)
		setDockerConfigLocation(t, tmpDir)
	}

	// Group teardown
	defer func() {
		localhostHTTPOnlyNoAuthStd.Cleanup()
		setDockerConfigLocation(t, old)
	}()

	testCases := []struct {
		url                string
		success            bool
		setInsecureToTrue  bool
		setInsecureToFalse bool
		username           string
		password           string
	}{
		// No port
		// - should always fail, as the default port is 443
		{
			// - should try: https://localhost:443
			url:     "localhost",
			success: false,
		},
		{
			// - should try: https://localhost:443
			url:                "localhost",
			success:            false,
			setInsecureToFalse: true,
		},
		{
			// - should try: https://localhost:443, https(no verify)://localhost:443, http://localhost:443,
			url:               "localhost",
			success:           false,
			setInsecureToTrue: true,
		},
		// Set port to 80
		// - should fail unless insecure
		{
			// - should try: https://localhost:80
			url:     "localhost:80",
			success: true,
		},
		{
			// - should try: https://localhost:80
			url:                "localhost:80",
			success:            false,
			setInsecureToFalse: true,
		},
		{
			// - should try: https://localhost:80, https(no verify)://localhost:80, http://localhost:80
			url:               "localhost:80",
			success:           true,
			setInsecureToTrue: true,
		},
		/*
			// Set port to 443
			// - should always fail
			{
				// - should try: https://localhost:443
				url:     "localhost:443",
				success: false,
			},
			{
				// - should try: https://localhost:443
				url:                "localhost:443",
				success:            false,
				setInsecureToFalse: true,
			},
			{
				// - should try: https://localhost:443, https(no verify)://localhost:443, http://localhost:443
				url:               "localhost:443",
				success:           false,
				setInsecureToTrue: true,
			},
			// Set port to 8080
			// - should always fail
			{
				// - should try: https://localhost:8080
				url:     "localhost:8080",
				success: false,
			},
			{
				// - should try: https://localhost:8080
				url:                "localhost:8080",
				success:            false,
				setInsecureToFalse: true,
			},
			{
				// - should try: https://localhost:8080, https(no verify)://localhost:8080, http://localhost:8080
				url:               "localhost:8080",
				success:           false,
				setInsecureToTrue: true,
			},
		*/
	}

	t.Run("Localhost login testing", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Login against %s (insecure set: %t - to true: %t - to false: %t)",
				tc.url, tc.setInsecureToTrue || tc.setInsecureToFalse, tc.setInsecureToTrue, tc.setInsecureToFalse), func(t *testing.T) {
				// t.Parallel()
				setup()
				username := tc.username
				password := tc.password
				if username == "" {
					username = "whatever"
				}
				if password == "" {
					password = "whatever"
				}
				args := []string{
					"--debug-full",
					"login",
					"--username", username,
					"--password", username,
				}
				if tc.setInsecureToTrue {
					args = append(args, "--insecure-registry=true")
				}
				if tc.setInsecureToFalse {
					args = append(args, "--insecure-registry=false")
				}

				args = append(args, tc.url)
				cmd := base.Cmd(args...)
				if tc.success {
					cmd.AssertOK()
				} else {
					cmd.AssertFail()
				}

				args[len(args)-1] = strings.Replace(tc.url, "localhost", "127.0.0.1", 1)
				cmd = base.Cmd(args...)
				if tc.success {
					cmd.AssertOK()
				} else {
					cmd.AssertFail()
				}
			})
		}
	})
}

func AAAATestLoginLocalhostNoAuthNNStd(t *testing.T) {
	base := testutil.NewBase(t)

	validUsername := "admin"
	validPassword := "password"

	// Group setup
	old := getDockerConfigLocation()
	// No auth, http registry, standard port
	localhostHTTPOnlyNoAuthStd := testregistry.NewRegistry(base, nil, 80, &testregistry.NoAuth{})
	localhostHTTPOnlyNoAuthNonStd := testregistry.NewRegistry(base, nil, 0, &testregistry.NoAuth{})
	localhostHTTPOnlyBasicAuthNonStd := testregistry.NewRegistry(base, nil, 0, &testregistry.BasicAuth{
		Username: validUsername,
		Password: validPassword,
	})

	setup := func() {
		tmpDir := createTempDir(t, 0700)
		setDockerConfigLocation(t, tmpDir)
	}

	// Group teardown
	defer func() {
		localhostHTTPOnlyNoAuthStd.Cleanup()
		localhostHTTPOnlyNoAuthNonStd.Cleanup()
		localhostHTTPOnlyBasicAuthNonStd.Cleanup()
		setDockerConfigLocation(t, old)
	}()

	testCases := []struct {
		description string
		url         string
		success     bool
		insecure    bool
		username    string
		password    string
	}{
		/**
		Standard http port
		*/
		{
			description: "Login against localhost:80",
			url:         "localhost:80",
			success:     true,
		},
		{
			description: "Login against localhost",
			url:         "localhost",
			success:     true,
		},
		{
			description: "Login against localhost:80",
			url:         "localhost:80",
			success:     true,
		},
		{
			description: "Login against localhost",
			url:         "localhost",
			// No port being provided, we default port to 443, so, we try to contact there, and nothing listens
			success: false,
		},
		{
			description: "Login against localhost",
			url:         "localhost",
			// No port being provided, we default port to 443, so, we try to contact there, and nothing listens
			// Since we pass insecure, should try to downgrade
			success:  false, // XXX FIXME
			insecure: true,
		},
		/*
			{
				description: "Login against https://localhost:80",
				url:         "localhost:80",
				// Server gave http response to https request - should never succeed
				success: false,
			},
			/**
			Non standard port
		*/
		{
			description: "Login against localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			success:     true,
		},
		{
			description: "Login against localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Defaults to assuming this is HTTPS, since the port is non-standard
			success: false,
		},
		{
			description: "Login against localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Defaults to assuming this is HTTPS, since the port is non-standard
			// Since we pass insecure, will retry with http, and succeed
			success:  true,
			insecure: true,
		},
		{
			description: "Login against https://localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Server gave http response to https request - should never succeed
			success: false,
		},
		/**
		Basic Auth
		*/
		{
			description: "Login against localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			success:     true,
		},
		{
			description: "Login against localhost:PORT",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Defaults to assuming this is HTTPS, since the port is non-standard
			success: false,
		},
		{
			description: "Login against localhost:PORT (invalid credentials)",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Defaults to assuming this is HTTPS, since the port is non-standard
			// Since we pass insecure, will retry with http, and fail on authentication
			success:  true,
			insecure: true,
		},
		{
			description: "Login against localhost:PORT (valid credentials)",
			url:         fmt.Sprintf("localhost:%d", localhostHTTPOnlyNoAuthNonStd.Port),
			// Defaults to assuming this is HTTPS, since the port is non-standard
			// Since we pass insecure, will retry with http, and succeed
			username: validUsername,
			password: validPassword,
			success:  true,
			insecure: true,
		},
	}

	t.Run("Localhost login testing", func(t *testing.T) {
		for _, tc := range testCases {
			t.Logf("Test case: %s", tc.description)

			setup()

			username := tc.username
			password := tc.password
			if username == "" {
				username = "whatever"
			}
			if password == "" {
				password = "whatever"
			}

			args := []string{
				"--debug-full",
				"login",
				"--username", username,
				"--password", username,
			}
			if tc.insecure {
				args = append(args, "--insecure-registry")
			}

			args = append(args, tc.url)
			cmd := base.Cmd(args...)
			if tc.success {
				cmd.AssertOK()
			} else {
				cmd.AssertFail()
			}

			args[len(args)-1] = strings.Replace(tc.url, "localhost", "127.0.0.1", 1)
			cmd = base.Cmd(args...)
			if tc.success {
				cmd.AssertOK()
			} else {
				cmd.AssertFail()
			}
			// XXX should test ipv6
		}
	})
}

/*
XXX with noauth, the client is still sending authentication tokens
iptables-save <- insane - most of these containers have disappeared - what are the rules doing here?
*/

/*
- login against localhost, no auth
- login against localhost, basic auth
- login against localhost, token auth

	type bearerAuthorizer struct {
		host   string
		bearer string
	}

	func (a *bearerAuthorizer) Authorize(ctx context.Context, req *http.Request) error {
		if req.Host != a.host {
			log.G(ctx).WithFields(log.Fields{
				"host":    req.Host,
				"cfgHost": a.host,
			}).Warn("Host doesn't match for bearer token")
			return nil
		}

		req.Header.Set("Authorization", "Bearer "+a.bearer)

		return nil
	}

	func (a *bearerAuthorizer) AddResponses(context.Context, []*http.Response) error {
		// Return not implemented to prevent retry of the request when bearer did not succeed
		return cerrdefs.ErrNotImplemented
	}
*/
func DDDTestLoginAgainstNoAuthRegistry(t *testing.T) {
}

func DDDTestLoginHTTPS(t *testing.T) {
	// Skip docker, because Docker doesn't have `--hosts-dir` option, and we don't want to contaminate the global /etc/docker/certs.d during this test
	testutil.DockerIncompatible(t)

	base := testutil.NewBase(t)

	// Credentials using outside of ascii range characters
	validUsername := "∞admin"
	validPassword := "∞validTestPassword"

	// Start a registry with a token auth server, over https
	ca := testca.New(base.T)
	authOverHTTPS := testregistry.NewAuthServer(base, ca, 0, validUsername, validPassword, true)
	defer authOverHTTPS.Cleanup()

	registryOverHTTPSRandomPort := testregistry.NewRegistry(base, ca, 0, authOverHTTPS.Auth)
	defer registryOverHTTPSRandomPort.Cleanup()

	registryOverHTTPS443 := testregistry.NewRegistry(base, ca, 443, authOverHTTPS.Auth)
	defer registryOverHTTPS443.Cleanup()

	testCases := []struct {
		regHost  string
		usePort  bool
		registry *testregistry.TestRegistry
		insecure bool
	}{
		{
			regHost:  "127.0.0.1",
			usePort:  true,
			registry: registryOverHTTPSRandomPort,
			insecure: false,
		},
		{
			regHost:  registryOverHTTPSRandomPort.IP.String(),
			usePort:  true,
			registry: registryOverHTTPSRandomPort,
			insecure: false,
		},
		{
			regHost:  "127.0.0.1",
			usePort:  false,
			registry: registryOverHTTPS443,
			insecure: false,
		},
		{
			regHost:  registryOverHTTPS443.IP.String(),
			usePort:  true,
			registry: registryOverHTTPS443,
			insecure: false,
		},
		{
			regHost:  "127.0.0.1",
			usePort:  false,
			registry: registryOverHTTPS443,
			insecure: false,
		},
		{
			regHost:  registryOverHTTPS443.IP.String(),
			usePort:  false,
			registry: registryOverHTTPS443,
			insecure: false,
		},
	}
	for _, tc := range testCases {
		username := validUsername
		password := validPassword
		//		for _, username := range []string{validUsername, "bogususername"} {
		//			for _, password := range []string{validPassword, "boguspassword"} {
		t.Logf("testing %v (%s:%s)", tc, username, password)
		shouldSucceed := username == validUsername && password == validPassword
		regHost := tc.registry.Scheme + "://" + tc.regHost
		if tc.usePort {
			regHost = fmt.Sprintf("%s:%d", regHost, tc.registry.Port)
		}
		var args []string
		if tc.insecure {
			args = append(args, "--insecure-registry")
		}
		args = append(args, "--debug-full", "--hosts-dir", tc.registry.HostsDir, "login", "-u", username, "-p", password, regHost)
		t.Log(strings.Join(args, " "))

		cmd := base.Cmd(args...)
		//fmt.Println(cmd.Out())
		//fmt.Println(cmd.Stdout)
		//fmt.Println(cmd.Stderr)
		if shouldSucceed {
			cmd.AssertOK()
		} else {
			cmd.AssertFail()
		}
		//			}
		//		}

	}
}

/*
func TestLoginWithSpecificRegHosts(t *testing.T) {
	// Skip docker, because Docker doesn't have `--hosts-dir` option, and we don't want to contaminate the global /etc/docker/certs.d during this test
	testutil.DockerIncompatible(t)

	base := testutil.NewBase(t)
	reg, auth := testregistry.NewHTTPS(base, "admin", "validTestPassword")
	defer reg.Cleanup()
	defer auth.Cleanup()

	regHost := net.JoinHostPort(reg.IP.String(), strconv.Itoa(reg.Port))

	t.Logf("Prepare regHost URL with path and Scheme")

	type testCase struct {
		url string
		log string
	}
	testCases := []testCase{
		{
			url: "" + path.Join(regHost, "test"),
			log: "Login with repository containing path and scheme in the URL",
		},
		{
			url: path.Join(regHost, "test"),
			log: "Login with repository containing path and without scheme in the URL",
		},
	}
	for _, tc := range testCases {
		t.Logf(tc.log)
		base.Cmd("--debug-full", "--hosts-dir", reg.HostsDir, "login", "-u", "admin", "-p", "validTestPassword", tc.url).AssertOK()
	}

}

/*
func TestLoginWithPlainHttp(t *testing.T) {
	testutil.DockerIncompatible(t)
	base := testutil.NewBase(t)
	reg5000, auth2 := testregistry.NewAuthWithHTTP(base, "admin", "validTestPassword", 0)
	reg80, auth := testregistry.NewAuthWithHTTP(base, "admin", "validTestPassword", 80)

	defer reg5000.Cleanup()
	defer reg80.Cleanup()
	defer auth.Cleanup()
	defer auth2.Cleanup()

	testCasesForPort5000 := []struct {
		regHost           string
		regPort           int
		useRegPort        bool
		username          string
		password          string
		shouldSuccess     bool
		registry          *testregistry.TestRegistry
		shouldUseInSecure bool
	}{
		{
			regHost:           "127.0.0.1",
			regPort:           5000,
			useRegPort:        true,
			username:          "admin",
			password:          "validTestPassword",
			shouldSuccess:     true,
			registry:          reg5000,
			shouldUseInSecure: true,
		},
		{
			regHost:           "127.0.0.1",
			regPort:           5000,
			useRegPort:        true,
			username:          "admin",
			password:          "invalidTestPassword",
			shouldSuccess:     false,
			registry:          reg5000,
			shouldUseInSecure: true,
		},
		{
			regHost:    "127.0.0.1",
			regPort:    5000,
			useRegPort: true,
			username:   "admin",
			password:   "validTestPassword",
			// Following the merging of the below, any localhost/loopback registries will
			// get automatically downgraded to HTTP so this will still succceed:
			// https://github.com/containerd/containerd/pull/7393
			// This comment is missing a point: the auth token ip address that will be returned by the registry
			// server is decided by a call to NonLoopbackIPv4. Thus, when redirecting, it is likely not localhost anymore
			// and the client will fail verifying the certificate
			// However, on tr
			shouldSuccess:     true,
			registry:          reg5000,
			shouldUseInSecure: true,
		},
		{
			regHost:           "127.0.0.1",
			regPort:           80,
			useRegPort:        false,
			username:          "admin",
			password:          "validTestPassword",
			shouldSuccess:     true,
			registry:          reg80,
			shouldUseInSecure: true,
		},
		{
			regHost:           "127.0.0.1",
			regPort:           80,
			useRegPort:        false,
			username:          "admin",
			password:          "invalidTestPassword",
			shouldSuccess:     false,
			registry:          reg80,
			shouldUseInSecure: true,
		},
		{
			regHost:    "127.0.0.1",
			regPort:    80,
			useRegPort: false,
			username:   "admin",
			password:   "validTestPassword",
			// Following the merging of the below, any localhost/loopback registries will
			// get automatically downgraded to HTTP so this will still succceed:
			// https://github.com/containerd/containerd/pull/7393
			shouldSuccess:     true,
			registry:          reg80,
			shouldUseInSecure: true,
		},
	}
	for _, tc := range testCasesForPort5000 {
		// tcName := fmt.Sprintf("%+v", tc)
		// t.Run(tcName, func(t *testing.T) {
		// /root/.docker/config.json

		regHost := tc.regHost
		if tc.useRegPort {
			regHost = fmt.Sprintf("%s:%d", regHost, tc.regPort)
		}
		if tc.shouldSuccess {
			t.Logf("Good password")
		} else {
			t.Logf("Bad password")
		}
		var args []string
		if tc.shouldUseInSecure {
			args = append(args, "--insecure-registry")
		}
		args = append(args, []string{
			"--debug-full", "--hosts-dir", tc.registry.HostsDir, "login", "-u", tc.username, "-p", tc.password, regHost,
		}...)
		t.Log(strings.Join(args, " "))

		cmd := base.Cmd(args...)
		if tc.shouldSuccess {
			cmd.AssertOK()
		} else {
			cmd.AssertFail()
		}

		// })
	}
}
*/
