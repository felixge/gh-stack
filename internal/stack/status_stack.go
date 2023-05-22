package stack

import (
	"bytes"
	"fmt"
)

type StatusStack struct {
	LocalStack   LocalStack
	RemoteStacks RemoteStacks
	StatusItems  []*StatusItem
}

func (s *StatusStack) Load(c *Context) error {
	if err := gitFetch(c); err != nil {
		return err
	}
	if err := s.LocalStack.Load(c); err != nil {
		return err
	}
	if err := s.RemoteStacks.Load(c, &s.LocalStack); err != nil {
		return err
	}

	uidLookup := map[string]*StatusItem{}
	for _, localCommit := range s.LocalStack.Commits {
		statusItem := &StatusItem{
			UID:         localCommit.UID,
			Oneline:     localCommit.Oneline(),
			LocalCommit: localCommit,
		}
		if localCommit.UID != "" {
			uidLookup[statusItem.UID] = statusItem
		}
		s.StatusItems = append(s.StatusItems, statusItem)
	}

	for _, remoteStack := range s.RemoteStacks.Stacks {
		statusItem, ok := uidLookup[remoteStack.UID]
		if !ok {
		} else if statusItem.RemoteStack != nil {
			return fmt.Errorf(
				"multiple remote stacks for local commit ref=%s uid=%s",
				statusItem.LocalCommit.Hash,
				statusItem.LocalCommit.UID,
			)
		} else {
			statusItem.RemoteStack = remoteStack
		}
	}

	return nil
}

func (s *StatusStack) String() string {
	var buf bytes.Buffer
	for _, item := range s.StatusItems {
		status := "new"
		if item.RemoteStack != nil {
			status = "modified"
		}
		fmt.Fprintf(&buf, "%s %s\n", status, item.Oneline)
	}
	return buf.String()
}

type StatusItem struct {
	UID         string
	Oneline     string
	LocalCommit *GitCommit
	RemoteStack *RemoteStack
	PullRequest *PullRequest
}
