/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime/trace"
	"strings"
	"sync"
	"text/template"

	"github.com/google/go-github/v52/github"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type pushFlags struct {
	DryRun  bool
	Remote  string
	Base    string
	Verbose bool
}

//
// func init() {
// 	fmt.Printf("\"init\": %v\n", "init")
// 	rootCmd.AddCommand(updateCmd)
//
// 	viper.BindPFlag("gitremote", updateCmd.Flags().Lookup("gitremote"))
// 	// cmd.Flags().StringVarP(&globalFlags.Config.BaseBranch, "base-branch", "b", "main", "base branch to target with pull requests")
// 	// initConfigFlags(updateCmd)
// }

func runPush(cmd *cobra.Command, _ []string, flags pushFlags) error {
	ctx := cmd.Context()
	baseBranch := fmt.Sprintf("%s/%s", flags.Remote, flags.Base)
	localCommits, err := gitLocalCommits(ctx, fmt.Sprintf("%s..HEAD", baseBranch))
	if err != nil {
		return err
	}

	dispatch := func(msg string, fn func() error) error {
		if flags.DryRun || flags.Verbose {
			cmd.Println(msg)
		}
		if !flags.DryRun {
			return fn()
		}
		return nil
	}

	localCommits, err = addStackCommitIDs(ctx, localCommits, dispatch, baseBranch)
	if err != nil {
		return err
	}

	remoteURL, err := gitRemoteURL(ctx, flags.Remote)
	if err != nil {
		return err
	}

	owner, repo, err := parseGitHubRemoteURL(remoteURL)
	if err != nil {
		return err
	}

	gh, err := initGitHubClient(ctx)
	if err != nil {
		return err
	}

	prs, err := githubFindPullRequests(ctx, gh, owner, repo, localCommits)
	if err != nil {
		return err
	}

	if err := gitForcePushCommits(ctx, flags.Remote, localCommits, prs, dispatch); err != nil {
		return err
	}

	prs, err = githubCreatePRs(ctx, gh, flags.Remote, owner, repo, flags.Base, localCommits, prs, dispatch)
	if err != nil {
		return err
	}

	if err := githubEditPRs(ctx, gh, owner, repo, flags.Base, localCommits, prs, dispatch); err != nil {
		return err
	}

	return nil
}

type githubConfigFile struct {
	Hosts map[string]githubConfig `yaml:",inline"`
}

type githubConfig struct {
	User        string `yaml:"user"`
	OAuthToken  string `yaml:"oauth_token"`
	GitProtocol string `yaml:"git_protocol"`
}

func loadGitHubConfig(filepath string) (*githubConfig, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config githubConfigFile
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	hostname := "github.com"
	host, ok := config.Hosts[hostname]
	if !ok {
		return nil, fmt.Errorf("hostname %q not found in: %s", hostname, filepath)
	}
	return &host, nil
}

type gitCommand struct {
	Args  []string
	Stdin io.Reader
}

func (g *gitCommand) Run(ctx context.Context) (string, error) {
	out := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "git", g.Args...)
	cmd.Stdin = g.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(g.Args, " "), err)
	}
	return strings.TrimSpace(out.String()), nil

}

func git(ctx context.Context, args ...string) (string, error) {
	return (&gitCommand{Args: args}).Run(ctx)
}

// gitRemoteURL returns the git remote URL for the given remote.
func gitRemoteURL(ctx context.Context, remote string) (string, error) {
	return git(ctx, "config", "--get", fmt.Sprintf("remote.%s.url", remote))
}

func gitForcePush(ctx context.Context, remote, localHash, remoteBranch string) error {
	ctx, task := trace.NewTask(ctx, "gitForcePush")
	defer task.End()

	_, err := git(ctx, "push", "-f", remote, fmt.Sprintf("%s:refs/heads/%s", localHash, remoteBranch))
	return err
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

var githubURLPattern = regexp.MustCompile(`(?:git@github\.com:|https://github\.com/)([^/]+)/(.+?)(?:\.git)?$`)

func getBaseRef(ctx context.Context, client *github.Client, owner, repo string) (string, error) {
	return "", nil
}

var _randomSeparator struct {
	sync.Once
	value string
}

func randomSeparator() (string, error) {
	var err error
	_randomSeparator.Do(func() {
		buf := make([]byte, 16)
		_, err = rand.Read(buf)
		if err != nil {
			panic(err)
		}
		_randomSeparator.value = "|" + hex.EncodeToString(buf) + "|"
	})
	return _randomSeparator.value, err
}

func gitLocalCommits(ctx context.Context, args ...string) ([]*localCommit, error) {
	ctx, task := trace.NewTask(ctx, "gitLocalCommits")
	defer task.End()

	sep, err := randomSeparator()
	if err != nil {
		return nil, err
	}
	logArgs := append([]string{"log", "--pretty=%H" + sep + "%B" + sep}, args...)
	out, err := git(ctx, logArgs...)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, sep)

	type result struct {
		index  int
		commit *localCommit
	}
	p := pool.NewWithResults[*result]().WithErrors()
	for i := 0; i < len(parts)-1; i += 2 {
		i := i
		p.Go(func() (*result, error) {
			hash := strings.TrimSpace(parts[i])
			msg := parts[i+1]

			stackCommitID, err := gitTrailerValue(ctx, msg, commitIDTrailerKey)
			if err != nil {
				return nil, err
			}
			return &result{
				index: i / 2,
				commit: &localCommit{
					Hash:          hash,
					Message:       msg,
					StackCommitID: stackCommitID,
				},
			}, nil
		})
	}
	results, err := p.Wait()
	if err != nil {
		return nil, err
	}
	commits := make([]*localCommit, len(results))
	for _, r := range results {
		commits[r.index] = r.commit
	}
	return commits, nil

}

type localCommit struct {
	// Hash is the git commit hash.
	Hash string
	// Message string.
	Message string
	// StackCommitID is the unique id of this commit used by gh-stack.
	StackCommitID string
}

func (c *localCommit) Oneline() string {
	line, _, _ := strings.Cut(c.Message, "\n")
	return line
}

func initGitHubClient(ctx context.Context) (*github.Client, error) {
	token, err := guessOAuthToken()
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

func gitSetTrailers(ctx context.Context, msg string, trailers ...string) (string, error) {
	cmd := gitCommand{
		Args:  []string{"interpret-trailers"},
		Stdin: strings.NewReader(msg),
	}
	for _, trailer := range trailers {
		cmd.Args = append(cmd.Args, "--if-exists", "replace", "--trailer", trailer)
	}
	return cmd.Run(ctx)
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[0:7]
	}
	return hash
}

const commitIDTrailerKey = "Stack-Commit-ID"

func stackCommitIDTrailer(commitHash string) string {
	return fmt.Sprintf("%s: %s", commitIDTrailerKey, stackCommitIDValue(commitHash))
}

func stackCommitIDValue(commitHash string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(commitHash)))
}

func gitTrailerValue(ctx context.Context, msg, trailerKey string) (string, error) {
	ctx, task := trace.NewTask(ctx, "gitTrailerValue")
	defer task.End()

	cmd := gitCommand{
		Args:  []string{"interpret-trailers", "--parse"},
		Stdin: strings.NewReader(msg),
	}
	trailers, err := cmd.Run(ctx)
	if err != nil {
		return "", err
	}
	var trailerValue string
	for _, line := range strings.Split(trailers, "\n") {
		parts := strings.Split(line, ": ")
		if len(parts) == 2 && parts[0] == trailerKey {
			trailerValue = parts[1]
		}
	}
	return trailerValue, nil
}

func addStackCommitIDs(
	ctx context.Context,
	commits []*localCommit,
	dispatch func(string, func() error) error,
	baseBranch string,
) ([]*localCommit, error) {
	ctx, task := trace.NewTask(ctx, "addStackCommitIDs")
	defer task.End()

	var rewordCommits []string
	for _, commit := range commits {
		if commit.StackCommitID != "" {
			continue
		}

		commit.StackCommitID = stackCommitIDValue(commit.Hash)
		dispatch(fmt.Sprintf(
			"reword commit %s to add %q trailer",
			shortHash(commit.Hash),
			stackCommitIDTrailer(commit.Hash),
		), func() error {
			rewordCommits = append(rewordCommits, commit.Hash)
			return nil
		})
	}

	if len(rewordCommits) == 0 {
		return commits, nil
	}

	commitFile, err := os.CreateTemp("", "gh-stack")
	if err != nil {
		return nil, err
	}
	defer commitFile.Close()
	defer os.RemoveAll(commitFile.Name())

	if _, err := commitFile.WriteString(strings.Join(rewordCommits, "\n")); err != nil {
		return nil, err
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	if _, err = git(
		ctx,
		// Disable git hooks for better performance. We're just appending a
		// trailer to the message, no hook should want to prevent that.
		"-c", "core.hooksPath=",
		// Use the rebase-edit-todo command to select the rewordCommits for
		// rewording.
		"-c", fmt.Sprintf("sequence.editor=%s rebase-edit-todo %s", exe, commitFile.Name()),
		// Use the rebase-add-trailer command to add the stack commit id
		// trailers to those commits.
		"-c", fmt.Sprintf("core.editor=%s rebase-add-trailer %s", exe, commitFile.Name()),
		"rebase", "-i", baseBranch, "--autostash",
	); err != nil {
		return nil, err
	}

	// Reload commit history after rewording
	return gitLocalCommits(ctx, fmt.Sprintf("%s..HEAD", baseBranch))
}

func stackCommitIDBranch(stackCommitID string) string {
	return fmt.Sprintf("gh-stack/%s", stackCommitID)
}

func gitForcePushCommits(
	ctx context.Context,
	remote string,
	localCommits []*localCommit,
	prs map[string]*github.PullRequest,
	dispatch func(string, func() error) error,
) error {
	ctx, task := trace.NewTask(ctx, "gitForcePushCommits")
	defer task.End()

	var branches []string
	for _, commit := range localCommits {
		pr, ok := prs[commit.Hash]
		if ok && pr.Head.GetSHA() == commit.Hash {
			continue
		}

		remoteBranch := stackCommitIDBranch(commit.StackCommitID)
		dispatch(fmt.Sprintf(
			"force push commit to %s to %s branch %s",
			shortHash(commit.Hash),
			remote,
			remoteBranch,
		), func() error {
			branches = append(branches, fmt.Sprintf("%s:refs/heads/%s", commit.Hash, remoteBranch))
			return nil
		})
	}

	if len(branches) == 0 {
		return nil
	}
	_, err := git(ctx, append([]string{"push", "-f", remote}, branches...)...)
	return err
}

func githubCreatePRs(
	ctx context.Context,
	gh *github.Client,
	remote string,
	owner string,
	repo string,
	baseBranch string,
	localCommits []*localCommit,
	existingPRs map[string]*github.PullRequest,
	dispatch func(string, func() error) error,
) (map[string]*github.PullRequest, error) {
	ctx, task := trace.NewTask(ctx, "githubCreatePRs")
	defer task.End()

	p := pool.NewWithResults[CommitWithPR]().WithMaxGoroutines(10).WithErrors()
	for i, commit := range localCommits {
		commit := commit
		prBase := baseBranch
		if i+1 < len(localCommits) {
			prBase = stackCommitIDBranch(localCommits[i+1].StackCommitID)
		}

		prTitle := commit.Oneline()
		prBody := "WIP"
		prHead := stackCommitIDBranch(commit.StackCommitID)

		if _, ok := existingPRs[commit.Hash]; !ok {
			dispatch(
				fmt.Sprintf("create PR for commit %s against %s branch %s", commit.Hash, remote, prBase),
				func() error {
					p.Go(func() (CommitWithPR, error) {
						ctx, task := trace.NewTask(ctx, "createPR")
						defer task.End()

						newPR := &github.NewPullRequest{
							Title: github.String(prTitle),
							Head:  github.String(prHead),
							Base:  github.String(prBase),
							Body:  github.String(prBody),
						}
						pr, _, err := gh.PullRequests.Create(ctx, owner, repo, newPR)
						return CommitWithPR{Commit: commit, PR: pr}, err
					})
					return nil
				},
			)
		}
	}

	results, err := p.Wait()
	if err != nil {
		return nil, err
	}
	combinedPRs := make(map[string]*github.PullRequest)
	for k, pr := range existingPRs {
		combinedPRs[k] = pr
	}
	for _, result := range results {
		combinedPRs[result.Commit.Hash] = result.PR
	}
	return combinedPRs, nil
}

func githubEditPRs(
	ctx context.Context,
	gh *github.Client,
	owner string,
	repo string,
	baseBranch string,
	localCommits []*localCommit,
	prs map[string]*github.PullRequest,
	dispatch func(string, func() error) error,
) error {
	ctx, task := trace.NewTask(ctx, "githubCreateAndEditPRs")
	defer task.End()

	var pullRequests []*github.PullRequest
	for _, commit := range localCommits {
		if pr, ok := prs[commit.Hash]; ok {
			pullRequests = append(pullRequests, pr)
		}
	}

	p := pool.New().WithMaxGoroutines(10).WithErrors()
	for i, commit := range localCommits {
		commit := commit
		prBase := baseBranch
		if i+1 < len(localCommits) {
			prBase = stackCommitIDBranch(localCommits[i+1].StackCommitID)
		}

		prTitle := commit.Oneline()
		buf := &bytes.Buffer{}
		if err := prTemplate.Execute(buf, map[string]interface{}{
			"Commit":       commit,
			"PullRequests": pullRequests,
		}); err != nil {
			return err
		}
		prBody := buf.String()
		// prHead := stackCommitIDBranch(commit.StackCommitID)

		pr, ok := prs[commit.Hash]
		if !ok || pr.Head.GetSHA() != commit.Hash ||
			pr.GetTitle() != prTitle ||
			pr.GetBody() != prBody ||
			pr.GetBase().GetRef() != prBase {

			prNumber := ""
			if ok {
				prNumber = fmt.Sprintf(" #%d", pr.GetNumber())
			}

			dispatch(
				fmt.Sprintf("edit PR%s for commit %s", prNumber, shortHash(commit.Hash)),
				func() error {
					p.Go(func() error {
						if !ok {
							// ok should only be false if we're in --dry-run mode. If we get
							// this far otherwise, something is very broken.
							return fmt.Errorf("bug: could not find PR for commit %s", commit.Hash)
						}

						ctx, task := trace.NewTask(ctx, "editPR")
						defer task.End()

						pr.Title = &prTitle
						pr.Body = &prBody
						pr.Base.Ref = &prBase
						_, _, err := gh.PullRequests.Edit(ctx, owner, repo, pr.GetNumber(), pr)
						if err != nil {
							return fmt.Errorf("edit PR: %w", err)
						}
						return nil
					})
					return nil
				},
			)
		}
	}
	return p.Wait()
}

var prTemplate = template.Must(template.New("foo").Parse(`{{.Commit.Message}}

---

Stack:

{{range .PullRequests}}* #{{.GetNumber}} {{if eq .Head.GetSHA $.Commit.Hash}} ⬅{{end}}
{{end}}


This stacked PR was created using [gh-stack](https://github.com/felixge/gh-stack).
`))

type CommitWithPR struct {
	Commit *localCommit
	PR     *github.PullRequest
}

var errPRNotFound = errors.New("pull request not found")

func githubFindPullRequests(
	ctx context.Context,
	gh *github.Client,
	owner string,
	repo string,
	commits []*localCommit) (map[string]*github.PullRequest, error) {
	ctx, task := trace.NewTask(ctx, "githubFindPullRequests")
	defer task.End()

	p := pool.NewWithResults[CommitWithPR]().WithMaxGoroutines(10).WithErrors()
	for _, commit := range commits {
		commit := commit
		p.Go(func() (CommitWithPR, error) {
			remoteBranch := stackCommitIDBranch(commit.StackCommitID)
			pr, err := findOpenPR(ctx, gh, owner, repo, remoteBranch)
			if err == errPRNotFound {
				err = nil
			}
			return CommitWithPR{PR: pr, Commit: commit}, err
		})
	}
	results, err := p.Wait()
	if err != nil {
		return nil, err
	}
	m := make(map[string]*github.PullRequest)
	for _, result := range results {
		if result.PR != nil {
			m[result.Commit.Hash] = result.PR
		}
	}
	return m, nil
}

func findOpenPR(ctx context.Context, gh *github.Client, owner, repo, head string) (*github.PullRequest, error) {
	ctx, task := trace.NewTask(ctx, "findOpenPR")
	defer task.End()

	pulls, _, err := gh.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State: "open",
		Head:  fmt.Sprintf("%s:%s", owner, head),
	})
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("list PRs: %w", err)
	} else if len(pulls) == 0 {
		return nil, errPRNotFound
	} else if len(pulls) > 1 {
		return nil, fmt.Errorf("found more than one PR")
	}
	return pulls[0], nil
}
