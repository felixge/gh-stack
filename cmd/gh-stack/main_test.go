package main

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
	config Config
}

func localRemoteRepo(t *testing.T) Config {
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
		cmds = append(cmds, createCommitCommands("C", "")...)
		cmds = append(cmds, createCommitCommands("D", "Unique-D")...)
		require.NoError(t, local.RunMulti(cmds...))

		_localRemoteRepo.config = (Config{CmdEnv: local}).WithDefaults()
	})
	return _localRemoteRepo.config
}
