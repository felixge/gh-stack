package stack

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func realMain() error {
	return prInfo("felixge", "spr-test")
}

func prInfo(owner, repo string) error {
	config, err := readGhCLIConfig()
	if err != nil {
		return fmt.Errorf("failed to read gh config: %w", err)
	}

	token := (*config)["github.com"].OauthToken
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	prs, _, err := client.PullRequests.List(ctx, owner, repo, nil)
	if err != nil {
		return err
	}
	fmt.Printf("prs: %v\n", prs)
	// repos, _, err := client.Repositories.List(ctx, "", nil)
	return nil
}
