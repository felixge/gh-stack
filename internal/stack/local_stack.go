package stack

type LocalStack struct {
	Commits []*GitCommit
}

// Load populates the local stack according to the config.
func (l *LocalStack) Load(c *Context) (err error) {
	l.Commits, err = GitLog(c.cmd, c.mergeBase+".."+c.config.LocalHead)
	return err
}
