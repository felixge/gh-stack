package stack

import (
	"os"

	"golang.org/x/exp/slog"
)

type ContextOptions struct {
	// Dir is the directory to operate in
	Dir string
	// Verbose enables verbose logging.
	Verbose bool
	// LoadConfig determines if the context should load its config from disk.
	LoadConfig bool
	// LoadGithubCredentials determines if the github credentials should be
	// loaded automatically.
	LoadGithub bool
}

func (o ContextOptions) NewContext() (*Context, error) {
	c := &Context{}
	c.cmd.Dir = o.Dir
	if o.Verbose {
		c.logLevel.Set(slog.LevelDebug)
	} else {
		c.logLevel.Set(slog.LevelInfo)
	}
	slogOpt := slog.HandlerOptions{Level: &c.logLevel}
	c.log = slog.New(slogOpt.NewTextHandler(os.Stdout))
	c.cmd.Logger = c.log

	if o.LoadConfig {
		if err := c.config.Load(c); err != nil {
			return nil, err
		}
	} else {
		c.config = c.config.WithDefaults()
	}
	if c.config.Verbose {
		c.logLevel.Set(slog.LevelDebug)
	}

	if o.LoadGithub && c.config.GithubOAuthToken == "" {
		gh, err := readGhCLIConfig()
		if err != nil {
			return nil, err
		}
		c.config.GithubOAuthToken = (*gh)[c.config.RemoteHost].OauthToken
	}

	var err error
	c.mergeBase, err = MergeBase(c.cmd, c.config.LocalHead, c.config.RemoteRef())
	if err != nil {
		return nil, err
	}
	c.log.Debug("merge base", "ref", c.mergeBase)
	return c, nil
}

type Context struct {
	config    Config
	cmd       CmdEnv
	log       *slog.Logger
	logLevel  slog.LevelVar
	mergeBase string
}
