package stack

type LocalStack struct {
	MergeBase string
	Commits   []*GitCommit
}

// Load populates the local stack according to the config.
func (l *LocalStack) Load(c Config) error {
	ref, err := MergeBase(c.CmdEnv, c.LocalRef, c.RemoteRef())
	if err != nil {
		return err
	}
	l.Commits, err = GitLog(c.CmdEnv, ref+".."+c.LocalRef)
	return err
}
