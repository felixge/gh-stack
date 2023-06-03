/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var updateFlags struct {
	DryRun bool
	Remote string
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		remoteURL, err := gitRemoteURL(cmd.Context(), updateFlags.Remote)
		if err != nil {
			return err
		}

		owner, repo, err := parseGitHubRemoteURL(remoteURL)
		if err != nil {
			return err
		}

		fmt.Printf("owner: %v\n", owner)
		fmt.Printf("repo: %v\n", repo)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	updateCmd.Flags().BoolVarP(&updateFlags.DryRun, "dry-run", "n", false, "Output the steps the command will take without executing them.")
	updateCmd.Flags().StringVarP(&updateFlags.Remote, "remote", "r", "origin", "Name of the git remote to interact with.")
}

// gitRemoteURL returns the git remote URL for the given remote.
func gitRemoteURL(ctx context.Context, remote string) (string, error) {
	stdout := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "git", "config", "--get", fmt.Sprintf("remote.%s.url", remote))
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git remote URL for %q: %w", updateFlags.Remote, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// parseGitHubRemoteURL returns the owner and repo for a remoteURL if it
// follows one of the formats below. GitHub Enterprise is not supported (yet).
//
// 1. https://github.com/owner/repo.git
// 2. git@github.com:owner/repo.git
func parseGitHubRemoteURL(remoteURL string) (owner, repo string, err error) {
	matches := githubURLPattern.FindStringSubmatch(remoteURL)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("invalid GitHub remote URL: %s", remoteURL)
	}
	return matches[1], matches[2], nil
}

var githubURLPattern = regexp.MustCompile(`(?:git@github\.com:|https://github\.com/)([^/]+)/(.+)\.git`)
