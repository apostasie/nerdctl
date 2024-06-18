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
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"gotest.tools/v3/assert"
)

func TestVolumeInspectContainsLabels(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	testVolume := testutil.Identifier(t)

	var tearDown = func() {
		base.Cmd("volume", "rm", "-f", testVolume).Run()
	}

	tearDown()
	t.Cleanup(tearDown)

	base.Cmd("volume", "create", testVolume, "--label", "foo1=baz1", "--label", "foo2=baz2").AssertOK()

	inspect := base.InspectVolume(testVolume)
	inspectNerdctlLabels := *inspect.Labels
	expected := make(map[string]string, 2)
	expected["foo1"] = "baz1"
	expected["foo2"] = "baz2"
	assert.DeepEqual(base.T, expected, inspectNerdctlLabels)
}

func TestVolumeInspectSize(t *testing.T) {
	testutil.DockerIncompatible(t)

	t.Parallel()

	base := testutil.NewBase(t)
	testVolume := testutil.Identifier(t)

	var tearDown = func() {
		base.Cmd("volume", "rm", "-f", testVolume).Run()
	}

	tearDown()
	t.Cleanup(tearDown)

	base.Cmd("volume", "create", testVolume).AssertOK()

	var size int64 = 1028
	createFileWithSize(t, testVolume, size)
	volumeWithSize := base.InspectVolume(testVolume, []string{"--size"}...)
	assert.Equal(t, volumeWithSize.Size, size)
}

func createFileWithSize(t *testing.T, volume string, bytes int64) {
	base := testutil.NewBase(t)
	v := base.InspectVolume(volume)
	token := make([]byte, bytes)
	_, err := rand.Read(token)
	assert.NilError(t, err)
	err = os.WriteFile(filepath.Join(v.Mountpoint, "test-file"), token, 0644)
	assert.NilError(t, err)
}
