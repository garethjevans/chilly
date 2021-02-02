package domain_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/plumming/dx/pkg/domain"
	"github.com/plumming/dx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestCanDetermineBranchName(t *testing.T) {
	dir, err := ioutil.TempDir("", "domain_test__TestCanDetermineBranchName")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	c := util.Command{
		Name: "git",
		Args: []string{"init", "-b", "master"},
		Dir:  dir,
	}
	output, err := c.RunWithoutRetry()
	assert.NoError(t, err)
	t.Log(output)

	bn, err := domain.CurrentBranchName(dir)
	assert.NoError(t, err)
	t.Log(bn)
	assert.Equal(t, "master", bn)
}

func TestCanStash(t *testing.T) {
	dir, err := ioutil.TempDir("", "domain_test__TestCanStash")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	c := util.Command{
		Name: "git",
		Args: []string{"init", "-b", "master"},
		Dir:  dir,
	}
	output, err := c.RunWithoutRetry()
	assert.NoError(t, err)
	t.Log(output)

	err = domain.ConfigCommitterInformation(dir, "test@test.com", "test user")
	assert.NoError(t, err)

	d1 := []byte("# domain_test__TestCanStash\n")
	err = ioutil.WriteFile(path.Join(dir, "README.md"), d1, 0600)
	assert.NoError(t, err)

	output, err = domain.Add(dir, "README.md")
	assert.NoError(t, err)
	t.Log(output)

	output, err = domain.Commit(dir, "Initial Commit")
	assert.NoError(t, err)
	t.Log(output)

	localChanges, err := domain.LocalChanges(dir)
	assert.NoError(t, err)
	assert.False(t, localChanges)

	d1 = []byte("hello\ngo\n")
	err = ioutil.WriteFile(path.Join(dir, "README.md"), d1, 0600)
	assert.NoError(t, err)

	localChanges, err = domain.LocalChanges(dir)
	assert.NoError(t, err)
	assert.True(t, localChanges)

	output, err = domain.Status(dir)
	assert.NoError(t, err)
	t.Log(output)

	output, err = domain.Stash(dir)
	assert.NoError(t, err)
	t.Log(output)

	localChanges, err = domain.LocalChanges(dir)
	assert.NoError(t, err)
	assert.False(t, localChanges)

	output, err = domain.StashPop(dir)
	assert.NoError(t, err)
	t.Log(output)

	localChanges, err = domain.LocalChanges(dir)
	assert.NoError(t, err)
	assert.True(t, localChanges)
}

func TestCanDetermineRemoteNames(t *testing.T) {
	type test struct {
		raw         string
		remote      string
		expectedURL string
	}

	tests := []test{
		{
			raw: `origin  https://github.com/garethjevans/chilly (fetch)
origin  https://github.com/garethjevans/chilly (push)
upstream        https://github.com/plumming/dx (fetch)
upstream        https://github.com/plumming/dx (push)`,
			remote:      "origin",
			expectedURL: "https://github.com/garethjevans/chilly",
		},
		{
			raw: `origin  https://github.com/garethjevans/chilly (fetch)
origin  https://github.com/garethjevans/chilly (push)
upstream        https://github.com/plumming/dx (fetch)
upstream        https://github.com/plumming/dx (push)`,
			remote:      "upstream",
			expectedURL: "https://github.com/plumming/dx",
		},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("TestCanDetermineRemoteNames-%s", tc.remote), func(t *testing.T) {
			url, err := domain.ExtractURLFromRemote(strings.NewReader(tc.raw), tc.remote)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedURL, url)
		})
	}
}
