package stack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPullRequest(t *testing.T) {
	c, err := (ContextOptions{LoadGithub: true}).NewContext()
	require.NoError(t, err)

	var pr PullRequest
	require.NoError(t, pr.LoadBranch(c, "foo"))
	//
	// ctx := context.Background()
	// ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	// tc := oauth2.NewClient(ctx, ts)
	// client := github.NewClient(tc)
	//
	// // list all repositories for the authenticated user
	// opts := github.PullRequestListOptions{Head: "DataDog:spr/854776e7"}
	// prs, _, err := client.PullRequests.List(ctx, "DataDog", "profiling-backend", &opts)
	// require.NoError(t, err)
	//
	// for _, pr := range prs {
	// 	fmt.Printf("pr.Head.GetRef(): %s\n", pr.Head.GetLabel())
	// }
}
