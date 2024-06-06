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

package testregistry

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/nettestutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/testca"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"
)

type TestRegistry struct {
	IP       net.IP
	Port     int
	Scheme   string
	ListenIP net.IP
	Cleanup  func()
	Logs     func()
	HostsDir string // contains "<HostIP>:<ListenPort>/hosts.toml"
}

type TestTokenAuth struct {
	IP       net.IP
	Port     int
	Scheme   string
	ListenIP net.IP
	Cleanup  func()
	Logs     func()
	Auth     Auth
	CertPath string
}

// Note: since port are DNAT-ed, just trying to listen on the port won't do anything
// Either inspect iptables or whatever nerdctl reports
// Neither approach are bulletproof unfortunately and might fail dependent on how network is set up
func nextAvailablePort(base *testutil.Base, start int) (int, error) {
	usedPorts := map[string]bool{}
	all := base.Cmd("container", "ls", "-a", "--format", "{{.Ports}}")
	all.AssertOK()
	scanner := bufio.NewScanner(strings.NewReader(all.Out()))
	for scanner.Scan() {
		port := strings.Split(scanner.Text(), "->")
		port = strings.Split(port[0], ":")
		usedPorts[port[1]] = true
	}

	for _, used := usedPorts[strconv.Itoa(start)]; used; _, used = usedPorts[strconv.Itoa(start)] {
		start++
	}

	return start, nil
}

func NewAuthServer(base *testutil.Base, ca *testca.CA, port int, user, pass string, tls bool) *TestTokenAuth {
	name := testutil.Identifier(base.T)
	// listen on 0.0.0.0 to enable 127.0.0.1
	listenIP := net.ParseIP("0.0.0.0")
	hostIP, err := nettestutil.NonLoopbackIPv4()
	assert.NilError(base.T, err)
	if port == 0 {
		port, err = nextAvailablePort(base, 5005)
		assert.NilError(base.T, err)
	}
	containerName := fmt.Sprintf("auth-%s-%d", name, port)

	// Prepare configuration file for authentication server
	// Details: https://github.com/cesanta/docker_auth/blob/1.7.1/examples/simple.yml
	configFile, err := os.CreateTemp("", "authconfig")
	assert.NilError(base.T, err)
	bpass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	assert.NilError(base.T, err)
	configFileName := configFile.Name()
	//tlsCert := ""
	//tlsKey := ""
	scheme := "http"
	if tls {
		scheme = "https"
		//tlsCert = "/auth/domain.crt"
		//tlsKey = "/auth/domain.key"
	}
	_, err = configFile.Write([]byte(fmt.Sprintf(`
server:
  addr: ":5100"
  certificate: "/auth/domain.crt"
  key: "/auth/domain.key"
token:
  issuer: "Acme auth server"
  expiration: 900
users:
  "%s":
    password: "%s"
acl:
  - match: {account: "%s"}
    actions: ["*"]
`, user, string(bpass), user)))
	assert.NilError(base.T, err)

	cert := ca.NewCert(hostIP.String())

	// Run authentication server
	cmd := base.Cmd("run",
		"-d",
		"-p", fmt.Sprintf("%s:%d:5100", listenIP, port),
		"--name", containerName,
		"-v", cert.CertPath+":/auth/domain.crt",
		"-v", cert.KeyPath+":/auth/domain.key",
		"-v", configFileName+":/config/auth_config.yml",
		testutil.DockerAuthImage,
		"/config/auth_config.yml")
	cmd.AssertOK()

	joined := net.JoinHostPort(hostIP.String(), strconv.Itoa(port))
	if _, err = nettestutil.HTTPGet(fmt.Sprintf("%s://%s/auth", scheme, joined), 30, true); err != nil {
		base.Cmd("rm", "-f", containerName).Run()
		base.T.Fatal(err)
	}

	return &TestTokenAuth{
		IP:       hostIP,
		Port:     port,
		Scheme:   scheme,
		ListenIP: listenIP,
		CertPath: cert.CertPath,
		Auth: &TokenAuth{
			Address:  scheme + "://" + net.JoinHostPort(hostIP.String(), strconv.Itoa(port)),
			CertPath: cert.CertPath,
		},
		Cleanup: func() {
			base.Cmd("rm", "-f", containerName).AssertOK()
			assert.NilError(base.T, cert.Close())
			assert.NilError(base.T, configFile.Close())
			os.Remove(configFileName)
			assert.NilError(base.T, cert.Close())
		},
		Logs: func() {
			base.T.Logf("%s: %q", containerName, base.Cmd("logs", containerName).Run().String())
		},
	}

}

type Auth interface {
	Params() []string
}

type NoAuth struct {
}

func (na *NoAuth) Params() []string {
	return []string{}
}

type TokenAuth struct {
	Address  string
	CertPath string
}

func (ta *TokenAuth) Params() []string {
	return []string{
		"--env", "REGISTRY_AUTH=token",
		"--env", "REGISTRY_AUTH_TOKEN_REALM=" + ta.Address + "/auth",
		"--env", "REGISTRY_AUTH_TOKEN_SERVICE=Docker registry",
		"--env", "REGISTRY_AUTH_TOKEN_ISSUER=Acme auth server",
		"--env", "REGISTRY_AUTH_TOKEN_ROOTCERTBUNDLE=/auth/domain.crt",
		"-v", ta.CertPath + ":/auth/domain.crt",
	}
}

type BasicAuth struct {
	Realm    string
	HtFile   string
	Username string
	Password string
}

func (ba *BasicAuth) Params() []string {
	if ba.Realm == "" {
		ba.Realm = "Basic Realm"
	}
	if ba.HtFile == "" && ba.Username != "" && ba.Password != "" {
		pass := ba.Password
		encryptedPass, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		tmpDir, _ := os.MkdirTemp("", "htpasswd")
		ba.HtFile = filepath.Join(tmpDir, "htpasswd")
		_ = os.WriteFile(ba.HtFile, []byte(fmt.Sprintf(`%s:%s`, ba.Username, string(encryptedPass[:]))), 0600)
	}
	ret := []string{
		"--env", "REGISTRY_AUTH=htpasswd",
		"--env", "REGISTRY_AUTH_HTPASSWD_REALM=" + ba.Realm,
		"--env", "REGISTRY_AUTH_HTPASSWD_PATH=/htpasswd",
	}
	if ba.HtFile != "" {
		ret = append(ret, "-v", ba.HtFile+":/htpasswd")
	}
	return ret
}

func NewRegistry(base *testutil.Base, ca *testca.CA, port int, auth Auth) *TestRegistry {
	name := testutil.Identifier(base.T)
	// listen on 0.0.0.0 to enable 127.0.0.1
	listenIP := net.ParseIP("0.0.0.0")
	hostIP, err := nettestutil.NonLoopbackIPv4()
	assert.NilError(base.T, err)
	if port == 0 {
		port, err = nextAvailablePort(base, 5001)
		assert.NilError(base.T, err)
	}
	assert.NilError(base.T, err)
	containerName := fmt.Sprintf("registry-%s-%d", name, port)

	args := []string{
		"run",
		"-d",
		"-p", fmt.Sprintf("%s:%d:5000", listenIP, port),
		"--name", containerName,
	}

	scheme := "http"
	var cert *testca.Cert
	if ca != nil {
		scheme = "https"
		cert = ca.NewCert(hostIP.String(), "127.0.0.1")
		args = append(args,
			"--env", "REGISTRY_HTTP_TLS_CERTIFICATE=/registry/domain.crt",
			"--env", "REGISTRY_HTTP_TLS_KEY=/registry/domain.key",
			"-v", cert.CertPath+":/registry/domain.crt",
			"-v", cert.KeyPath+":/registry/domain.key",
		)
	}

	args = append(args, auth.Params()...)

	args = append(args, testutil.RegistryImage)
	cmd := base.Cmd(args...)
	cmd.AssertOK()

	if _, err = nettestutil.HTTPGet(fmt.Sprintf("%s://127.0.0.1:%s/v2", scheme, strconv.Itoa(port)), 30, true); err != nil {
		base.Cmd("rm", "-f", containerName).Run()
		base.T.Fatal(err)
	}

	hostsDir, err := os.MkdirTemp(base.T.TempDir(), "certs.d")
	assert.NilError(base.T, err)

	if ca != nil {
		joined := net.JoinHostPort(hostIP.String(), strconv.Itoa(port))
		hostsSubDir := filepath.Join(hostsDir, joined)
		err = os.MkdirAll(hostsSubDir, 0700)
		assert.NilError(base.T, err)
		hostsTOMLPath := filepath.Join(hostsSubDir, "hosts.toml")
		// See https://github.com/containerd/containerd/blob/main/docs/hosts.md
		hostsTOML := fmt.Sprintf(`
server = "https://%s"
[host."https://%s"]
  ca = %q
		`, joined, joined, ca.CertPath)
		base.T.Logf("Writing %q: %q", hostsTOMLPath, hostsTOML)
		err = os.WriteFile(hostsTOMLPath, []byte(hostsTOML), 0700)
		assert.NilError(base.T, err)

		joined = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		hostsSubDir = filepath.Join(hostsDir, joined)
		err = os.MkdirAll(hostsSubDir, 0700)
		assert.NilError(base.T, err)
		hostsTOMLPath = filepath.Join(hostsSubDir, "hosts.toml")
		hostsTOML = fmt.Sprintf(`
server = "https://%s"
[host."https://%s"]
  ca = %q
		`, joined, joined, ca.CertPath)
		base.T.Logf("Writing %q: %q", hostsTOMLPath, hostsTOML)
		err = os.WriteFile(hostsTOMLPath, []byte(hostsTOML), 0700)
		assert.NilError(base.T, err)
	}

	return &TestRegistry{
		IP:       hostIP,
		Port:     port,
		Scheme:   scheme,
		ListenIP: listenIP,
		Cleanup: func() {
			base.Cmd("rm", "-f", containerName).AssertOK()
			if cert != nil {
				assert.NilError(base.T, cert.Close())
			}
		},
		Logs: func() {
			base.T.Logf("%s: %q", containerName, base.Cmd("logs", containerName).Run().String())
		},
		HostsDir: hostsDir,
	}

}

func NewAuthWithHTTP(base *testutil.Base, user, pass string, port int) (*TestRegistry, *TestTokenAuth) {
	ca := testca.New(base.T)
	as := NewAuthServer(base, ca, 0, user, pass, false)
	auth := &TokenAuth{
		Address:  as.Scheme + "://" + net.JoinHostPort(as.IP.String(), strconv.Itoa(as.Port)),
		CertPath: as.CertPath,
	}
	re := NewRegistry(base, nil, port, auth)
	return re, as
}

func NewPlainHTTP(base *testutil.Base, listenPort int) *TestRegistry {
	return NewRegistry(base, nil, listenPort, &NoAuth{})
}

func NewHTTPS(base *testutil.Base, user, pass string) (*TestRegistry, *TestTokenAuth) {
	ca := testca.New(base.T)
	as := NewAuthServer(base, ca, 0, user, pass, true)
	auth := &TokenAuth{
		Address:  as.Scheme + "://" + net.JoinHostPort(as.IP.String(), strconv.Itoa(as.Port)),
		CertPath: as.CertPath,
	}
	re := NewRegistry(base, ca, 0, auth)
	return re, as
}
