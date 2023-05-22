package stack

type RemoteStacks struct {
	Stacks []*RemoteStack
}

func (r *RemoteStacks) Load(c *Context, ls *LocalStack) error {
	r.Stacks = []*RemoteStack{}
	for _, localCommit := range ls.Commits {
		if localCommit.UID == "" {
			// skip commits without UID
			continue
		}

		branch := localCommit.Branch()
		remoteCommits, err := GitLog(c.cmd, c.mergeBase+".."+c.config.RemoteName+"/"+branch)
		if err != nil {
			// the remote branch does not exist
			continue
		}
		r.Stacks = append(r.Stacks, &RemoteStack{
			UID:     localCommit.UID,
			Branch:  branch,
			Commits: remoteCommits,
		})
	}
	return nil
}

type RemoteStack struct {
	UID     string
	Branch  string
	Commits []*GitCommit
}
