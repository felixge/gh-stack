/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rebaseAddTrailerCmd represents the rebaseAddTrailer command
var rebaseAddTrailerCmd = &cobra.Command{
	Use:    "rebase-add-trailer",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("expected 2 args, but got: %d", len(args))
		}
		commitFile, messageFile := args[0], args[1]
		ctx := cmd.Context()

		// Load commits that should be reworded from commits file.
		commits, err := loadCommits(args[0])
		if err != nil {
			return err
		}

		// Add commit id trailer to message file.
		commit := commits[len(commits)-1]
		if err := addCommitIDTrailer(ctx, messageFile, commit); err != nil {
			return err
		}

		// Remove commit from list and write remaining commits back to the commit file.
		return writeCommits(commitFile, commits[0:len(commits)-1])
	},
}

func init() {
	rootCmd.AddCommand(rebaseAddTrailerCmd)
}

func addCommitIDTrailer(ctx context.Context, commitFile string, commit string) error {
	oldMsg, err := os.ReadFile(commitFile)
	if err != nil {
		return err
	}

	newMsg, err := gitSetTrailers(ctx, string(oldMsg), commitIDTrailer(commit))
	if err != nil {
		return err
	}

	return os.WriteFile(commitFile, []byte(newMsg), 0)
}

func writeCommits(commitFile string, commits []string) error {
	data := []byte(strings.Join(commits, "\n"))
	return os.WriteFile(commitFile, data, 0)
}
