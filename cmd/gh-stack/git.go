package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func GitLog(env CmdEnv, args ...string) ([]*GitCommit, error) {
	sep, err := randomSeparator()
	if err != nil {
		return nil, err
	}
	cmd := append([]string{"git", "log", "--pretty=%H" + sep + "%B" + sep}, args...)
	out, err := env.Run(cmd...)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	parts := strings.Split(out, sep)
	var commits []*GitCommit
	for i := 0; i < len(parts)-1; i += 2 {
		hash := strings.TrimSpace(parts[i])
		msg := strings.TrimSpace(parts[i+1])
		uid, err := ParseCommitUID(msg)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", hash, err)
		}

		commits = append(commits, &GitCommit{
			Hash:    hash,
			UID:     uid,
			Message: msg,
		})
	}
	return commits, nil
}

type GitCommit struct {
	// Hash is the git commit hash.
	Hash string
	// UID is the value of the Commit-UID trailer.
	UID string
	// Message string
	Message string
}

func (g GitCommit) Oneline() string {
	return strings.Split(g.Message, "\n")[0]
}

func ParseCommitUID(input string) (string, error) {
	commitPattern := regexp.MustCompile(`(?m)^Commit-UID:\s*(.*)$`)
	matches := commitPattern.FindAllStringSubmatch(input, -1)

	if matches == nil {
		return "", nil
	}

	if len(matches) > 1 {
		return "", errors.New("multiple Commit-UID trailers")
	}

	return matches[0][1], nil
}
