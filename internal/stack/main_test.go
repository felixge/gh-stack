package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var cleanups struct {
	mu  sync.Mutex
	fns []func() error
}

func appendCleanup(fn func() error) {
	cleanups.mu.Lock()
	defer cleanups.mu.Unlock()
	cleanups.fns = append(cleanups.fns, fn)
}

func TestMain(m *testing.M) {
	code := m.Run()
	cleanups.mu.Lock()
	defer cleanups.mu.Unlock()
	for _, fn := range cleanups.fns {
		if err := fn(); err != nil {
			fmt.Printf("cleanup error: %s", err)
		}
	}
	os.Exit(code)
}

var _localRemoteRepo struct {
	sync.Once
	ctx *Context
}

func localRemoteRepo(t *testing.T) *Context {
	t.Helper()
	_localRemoteRepo.Do(func() {
		env, cleanup := tmpCmdEnv(t)
		appendCleanup(cleanup)

		_, err := env.Run("mkdir", "-p", "remote")
		require.NoError(t, err)

		remote := env
		remote.Dir = filepath.Join(remote.Dir, "remote")

		cmds := [][]string{{"git", "init"}}
		cmds = append(cmds, createCommitCommands("A", "")...)
		cmds = append(cmds, createCommitCommands("B", "")...)
		require.NoError(t, remote.RunMulti(cmds...))

		_, err = env.Run("git", "clone", "./remote", "local")
		require.NoError(t, err)

		local := env
		local.Dir = filepath.Join(local.Dir, "local")

		cmds = nil
		cmds = append(cmds, createCommitCommands("C", "uid-c")...)
		cmds = append(cmds, []string{"git", "tag", "C"})
		cmds = append(cmds, createCommitCommands("D", "uid-d")...)
		cmds = append(cmds, []string{"git", "tag", "D"})
		cmds = append(cmds, createCommitCommands("E", "uid-e")...)
		cmds = append(cmds, createCommitCommands("F", "")...)
		require.NoError(t, local.RunMulti(cmds...))

		_, err = local.Run("git", "push", "origin", "C:refs/heads/gh-stack-commit-uid-c")
		require.NoError(t, err)
		_, err = local.Run("git", "push", "origin", "D:refs/heads/gh-stack-commit-uid-d")
		require.NoError(t, err)

		ctx, err := ContextOptions{Dir: local.Dir, Verbose: true}.NewContext()
		require.NoError(t, err)
		_localRemoteRepo.ctx = ctx
	})
	return _localRemoteRepo.ctx
}
