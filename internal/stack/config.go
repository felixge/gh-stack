package stack

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Verbose enables verbose logging, defaults to false.
	Verbose bool `yaml:"verbose"`
	// LocalHead is a git reference to the local top of the stack, defaults to "HEAD".
	LocalHead string `yaml:"local_head"`
	// RemoteHost is the name of the remote host, defaults to "github.com".
	RemoteHost string `yaml:"remote_host"`
	// RemoteName is the name of the remote repository to target,
	// defaults to "origin".
	RemoteName string `yaml:"remote_name"`
	// RemoteHead is the name of the remote branch to target,
	// defaults to "main".
	RemoteHead string `yaml:"remote_head"`
	// GithubOAuthToken holds the github OAuth token. Uses the gh cli token if
	// available.
	GithubOAuthToken string `yaml:""`
	// RemoteOwner is the name of the owner (user or org) of the remote
	// repository.
	RemoteOwner string `yaml:"remote_host"`
	// RemoteRepo is the name of the remote repository.
	RemoteRepo string `yaml:"remote_host"`
}

func (c *Config) Load(ctx *Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	gitRoot, err := gitRootDir(ctx)
	if err != nil {
		return err
	}

	*c = c.WithDefaults()
	for _, dir := range []string{home, gitRoot} {
		path := filepath.Join(dir, ".gh-stack.yml")
		data, err := ioutil.ReadFile(path)
		if os.IsNotExist(err); err != nil {
			ctx.log.Debug("config file does not exist, skipping", "path", path)
			continue
		} else if err != nil {
			return err
		}

		ctx.log.Debug("loaded config file", "path", path)
		if err := yaml.Unmarshal(data, c); err != nil {
			return err
		}
	}
	ctx.log.Debug("final config", "config", slog.AnyValue(c))

	return nil
}

func (c Config) WithDefaults() Config {
	if c.LocalHead == "" {
		c.LocalHead = "HEAD"
	}
	if c.RemoteHost == "" {
		c.RemoteHost = "github.com"
	}
	if c.RemoteName == "" {
		c.RemoteName = "origin"
	}
	if c.RemoteHead == "" {
		c.RemoteHead = "main"
	}
	return c
}

func (c Config) RemoteRef() string {
	return c.RemoteName + "/" + c.RemoteHead
}
