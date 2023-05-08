package stack

type RemoteStacks struct {
	Stacks []*RemoteStack
}

type RemoteStack struct {
	Commits []*RemoteCommit
}

type RemoteCommit struct {
}
