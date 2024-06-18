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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/containerd/log"
	"github.com/containerd/nerdctl/v2/pkg/buildkitutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil"
)

func TestSystemPrune(t *testing.T) {
	testutil.RequiresBuild(t)

	// This is pending https://github.com/containerd/nerdctl/pull/3096
	// Right now, networks are not namespaced, making this unsafe to parallelize
	// t.Parallel()

	base := testutil.NewBaseWithNamespace(t, testutil.Identifier(t))
	tID := testutil.Identifier(t)

	var tearDown = func() {
		base.Cmd("network", "rm", tID).Run()
		base.Cmd("volume", "rm", tID).Run()
		base.Cmd("rm", "-f", tID).Run()
	}

	tearDown()
	t.Cleanup(tearDown)

	base.Cmd("container", "prune", "-f").AssertOK()
	base.Cmd("network", "prune", "-f").AssertOK()
	base.Cmd("volume", "prune", "-f").AssertOK()
	base.Cmd("image", "prune", "-f", "--all").AssertOK()

	base.Cmd("network", "create", tID).AssertOK()
	base.Cmd("volume", "create", tID).AssertOK()
	vID := base.Cmd("volume", "create").Out()
	t.Cleanup(func() {
		base.Cmd("volume", "rm", vID).Run()
	})

	base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID), "--net", tID,
		"--name", tID, testutil.CommonImage).AssertOK()

	base.Cmd("ps", "-a").AssertOutContains(tID)
	base.Cmd("images").AssertOutContains(testutil.ImageRepo(testutil.CommonImage))

	base.Cmd("system", "prune", "-f", "--volumes", "--all").AssertOK()
	base.Cmd("volume", "ls").AssertOutContains(tID) // docker system prune --all --volume does not prune named volume
	base.Cmd("volume", "ls").AssertNoOut(vID)       // docker system prune --all --volume prune anonymous volume
	base.Cmd("ps", "-a").AssertNoOut(tID)
	base.Cmd("network", "ls").AssertNoOut(tID)
	base.Cmd("images").AssertNoOut(testutil.ImageRepo(testutil.CommonImage))

	if testutil.GetTarget() != testutil.Nerdctl {
		t.Skip("test skipped for buildkitd is not available with docker-compatible tests")
	}

	testutil.RequireExecutable(t, "buildctl")

	buildctlBinary, err := buildkitutil.BuildctlBinary()
	if err != nil {
		t.Fatal(err)
	}
	host, err := buildkitutil.GetBuildkitHost(testutil.Namespace)
	if err != nil {
		t.Fatal(err)
	}

	buildctlArgs := buildkitutil.BuildctlBaseArgs(host)
	buildctlArgs = append(buildctlArgs, "du")
	log.L.Debugf("running %s %v", buildctlBinary, buildctlArgs)
	buildctlCmd := exec.Command(buildctlBinary, buildctlArgs...)
	buildctlCmd.Env = os.Environ()
	stdout := bytes.NewBuffer(nil)
	buildctlCmd.Stdout = stdout
	if err := buildctlCmd.Run(); err != nil {
		t.Fatal(err)
	}
	readAll, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readAll), "Total:\t\t0B") {
		t.Errorf("buildkit cache is not pruned: %s", string(readAll))
	}
}
