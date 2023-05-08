package stack

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeBase(t *testing.T) {
	config := localRemoteRepo(t)

	ref, err := MergeBase(config.CmdEnv, "HEAD", "origin/main")
	require.NoError(t, err)

	commit, err := config.Run("git", "show", "-q", ref, "--pretty=format:%s")
	require.NoError(t, err)
	require.Equal(t, "B", commit)
}

func createCommitCommands(commit, uid string) (cmds [][]string) {
	msg := commit + "\n\nThis is commit: " + commit
	if uid != "" {
		msg += "\nCommit-UID: " + uid + "\n"
	}
	return [][]string{
		{"touch", commit},
		{"git", "add", commit},
		{"git", "commit", "--no-verify", "-m", msg},
	}
}

func tmpCmdEnv(t *testing.T) (cmdEnv CmdEnv, cleanup func() error) {
	t.Helper()
	tmpDir, err := ioutil.TempDir("", "gh-stack")
	require.NoError(t, err)

	cmdEnv.Dir = tmpDir
	cleanup = func() error { return os.RemoveAll(tmpDir) }
	return
}
