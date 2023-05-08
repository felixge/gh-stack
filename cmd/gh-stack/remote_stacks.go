package main

type RemoteStacks struct {
	Stacks []*RemoteStack
}

type RemoteStack struct {
	Commits []*RemoteCommit
}

type RemoteCommit struct {
}
