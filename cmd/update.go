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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

var updateFlags struct {
	DryRun     bool
	Remote     string
	BaseBranch string
}

var updateCmd = &cobra.Command{
	Use:          "update",
	Short:        "Pushes the local commit stack and creates/updates PRs for it.",
	Long:         ``,
	SilenceUsage: true,
	RunE:         runUpdate,
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	ctx, task := trace.NewTask(cmd.Context(), "runUpdate")
	defer task.End()

	baseBranch := fmt.Sprintf("%s/%s", updateFlags.Remote, updateFlags.BaseBranch)
	localCommits, err := gitLocalCommits(ctx, fmt.Sprintf("%s..HEAD", baseBranch))
	if err != nil {
		return err
	}

	dispatch := func(msg string, fn func() error) error {
		if updateFlags.DryRun || globalFlags.Verbose {
			cmd.Println(msg)
		}
		if !updateFlags.DryRun {
			return fn()
		}
		return nil
	}

	localCommits, err = addStackCommitIDs(ctx, localCommits, dispatch, baseBranch)
	if err != nil {
		return err
	}

	remoteURL, err := gitRemoteURL(ctx, updateFlags.Remote)
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

	if err := gitForcePushCommits(ctx, updateFlags.Remote, localCommits, prs, dispatch); err != nil {
		return err
	}

	// base := updateFlags.BaseBranch
	// for i := len(localCommits) - 1; i >= 0; i-- {
	// 	commit := localCommits[i]
	// 	pr, ok := prs[commit.Hash]
	// 	remoteBranch := stackCommitIDBranch(commit.StackCommitID)
	//
	// 	// Force-push branch if needed
	// 	if !ok || pr.Head.GetSHA() != commit.Hash {
	// 		if err := dispatch(fmt.Sprintf(
	// 			"force push commit to %s to %s branch %s",
	// 			shortHash(commit.Hash),
	// 			updateFlags.Remote,
	// 			remoteBranch,
	// 		), func() error {
	// 			return gitForcePush(ctx, updateFlags.Remote, commit.Hash, remoteBranch)
	// 		}); err != nil {
	// 			return err
	// 		}
	// 	}
	//
	// 	var prData = struct {
	// 		Title string
	// 		Body  string
	// 		Base  string
	// 		Head  string
	// 	}{commit.Oneline(), commit.Message, base, remoteBranch}
	//
	// 	buf := &bytes.Buffer{}
	// 	if err := prTemplate.Execute(buf, &prData); err != nil {
	// 		return err
	// 	}
	// 	prData.Body = buf.String()
	//
	// 	// Create or update PR if needed
	// 	if !ok {
	// 		if err := dispatch(
	// 			fmt.Sprintf("create PR for commit %s against %s branch %s", commit.Hash, updateFlags.Remote, base),
	// 			func() error {
	// 				newPR := &github.NewPullRequest{
	// 					Title: github.String(prData.Title),
	// 					Head:  github.String(prData.Head),
	// 					Base:  github.String(prData.Base),
	// 					Body:  github.String(prData.Body),
	// 				}
	// 				_, _, err := gh.PullRequests.Create(ctx, owner, repo, newPR)
	// 				if err != nil {
	// 					return fmt.Errorf("create PR: %w", err)
	// 				}
	// 				return nil
	// 			}); err != nil {
	// 			return err
	// 		}
	// 	} else if pr.Head.GetSHA() != commit.Hash ||
	// 		pr.GetTitle() != prData.Title ||
	// 		pr.GetBody() != prData.Body ||
	// 		pr.GetBase().GetRef() != prData.Base {
	// 		if err := dispatch(
	// 			fmt.Sprintf("edit PR for commit %s", shortHash(commit.Hash)),
	// 			func() error {
	// 				pr.Title = &prData.Title
	// 				pr.Body = &prData.Body
	// 				pr.Base.Ref = &prData.Base
	// 				_, _, err := gh.PullRequests.Edit(ctx, owner, repo, pr.GetNumber(), pr)
	// 				if err != nil {
	// 					return fmt.Errorf("edit PR: %w", err)
	// 				}
	// 				return nil
	// 			}); err != nil {
	// 			return err
	// 		}
	// 	}
	//
	// 	base = remoteBranch
	// }
	return nil
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
	updateCmd.Flags().StringVarP(&updateFlags.BaseBranch, "base", "b", "main", "Name of the base branch to target with pull requests.")
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

	p := pool.NewWithResults[*localCommit]().WithErrors()

	for i := 0; i < len(parts)-1; i += 2 {
		i := i
		p.Go(func() (*localCommit, error) {
			hash := strings.TrimSpace(parts[i])
			msg := parts[i+1]

			stackCommitID, err := gitTrailerValue(ctx, msg, commitIDTrailerKey)
			if err != nil {
				return nil, err
			}
			return &localCommit{
				Hash:          hash,
				Message:       msg,
				StackCommitID: stackCommitID,
			}, nil
		})
	}
	return p.Wait()
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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	githubConfig, err := loadGitHubConfig(filepath.Join(home, ".config", "gh", "hosts.yml"))
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubConfig.OAuthToken})
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

var prTemplate = template.Must(template.New("foo").Parse(`{{.Body}}

---


This stacked PR was created using [gh-stack](https://github.com/felixge/gh-stack).
`))
