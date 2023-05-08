package main

import "strings"

func MergeBase(env CmdEnv, refA, refB string) (string, error) {
	out, err := env.Run("git", "merge-base", refA, refB)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
