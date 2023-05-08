package stack

type Config struct {
	CmdEnv
	// LocalRef is the name of the local ref, defaults to "HEAD".
	LocalRef string
	// RemoteName is the name of the remote repository to target,
	// defaults to "origin".
	RemoteName string
	// RemoteName is the name of the branch in the remote repository to target. Defaults to "main".
	RemoteBranch string
}

func (c *Config) Load(env CmdEnv) error {
	c.CmdEnv = env
	return nil
}

func (c Config) WithDefaults() Config {
	if c.LocalRef == "" {
		c.LocalRef = "HEAD"
	}
	if c.RemoteName == "" {
		c.RemoteName = "origin"
	}
	if c.RemoteBranch == "" {
		c.RemoteBranch = "main"
	}
	return c
}

func (c Config) RemoteRef() string {
	return c.RemoteName + "/" + c.RemoteBranch
}
