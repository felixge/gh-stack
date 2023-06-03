/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v52/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// prCmd represents the pr command
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GH_TOKEN")},
		)
		tc := oauth2.NewClient(ctx, ts)

		client := github.NewClient(tc)

		pr := PullRequest{
			Title: fmt.Sprintf("Time: %s", time.Now().Format(time.RFC3339)),
			Head:  "foo",
			Base:  "main",
			Body:  "body *is* cool",
			Owner: "felixge",
			Repo:  "gh-stack-test",
		}

		pull, err := createOrUpdatePR(ctx, client, pr)
		if err != nil {
			return err
		}
		fmt.Printf("pull.HTMLURL: %v\n", pull.GetHTMLURL())

		return nil
	},
}

type PullRequest struct {
	Title string
	Head  string
	Base  string
	Body  string

	Owner string
	Repo  string
}

func createOrUpdatePR(ctx context.Context, client *github.Client, pr PullRequest) (*github.PullRequest, error) {
	// Find existing PR for the same head (branch)
	pulls, _, err := client.PullRequests.List(ctx, pr.Owner, pr.Repo, &github.PullRequestListOptions{
		State: "open",
		Head:  fmt.Sprintf("%s:%s", pr.Owner, pr.Head),
	})
	if err != nil {
		// Handle error
		return nil, err
	}

	// Create new PR if there isn't an existing one to update
	if len(pulls) == 0 {
		newPR := &github.NewPullRequest{
			Title: github.String(pr.Title),
			Head:  github.String(pr.Head),
			Base:  github.String(pr.Base),
			Body:  github.String(pr.Body),
		}
		pull, _, err := client.PullRequests.Create(ctx, pr.Owner, pr.Repo, newPR)
		return pull, err
	}

	// We should never see more than one PR for the same head, return an error if
	// this happens somehow anyway.
	if len(pulls) != 1 {
		return nil, fmt.Errorf("expected 0 or 1 PR, but got %d", len(pulls))
	}

	// Update existing PR
	pull := pulls[0]
	pull.Title = github.String(pr.Title)
	pull.Body = github.String(pr.Body)
	pull.Base.Ref = github.String(pr.Base)
	pull, _, err = client.PullRequests.Edit(ctx, pr.Owner, pr.Repo, pull.GetNumber(), pull)
	return pull, err
}

func init() {
	rootCmd.AddCommand(prCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// prCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// prCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
