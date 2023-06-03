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
	"strings"
	"sync"

	"github.com/google/go-github/v52/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

var updateFlags struct {
	DryRun     bool
	Remote     string
	BaseBranch string
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
		ctx := cmd.Context()
		baseBranch := fmt.Sprintf("%s/%s", updateFlags.Remote, updateFlags.BaseBranch)
		commits, err := localCommits(ctx, fmt.Sprintf("%s..HEAD", baseBranch))
		if err != nil {
			return err
		}

		if err := addStackCommitIDs(ctx, commits, cmd.OutOrStdout(), baseBranch); err != nil {
			return err
		}

		for _, gc := range commits {
			fmt.Printf("gc.StackCommitID: %v\n", gc.StackCommitID)
		}

		remoteURL, err := gitRemoteURL(ctx, updateFlags.Remote)
		if err != nil {
			return err
		}

		owner, repo, err := parseGitHubRemoteURL(remoteURL)
		if err != nil {
			return err
		}

		ghClient, err := initGitHubClient(ctx)
		if err != nil {
			return err
		}

		ref, err := getBaseRef(ctx, ghClient, owner, repo)
		if err != nil {
			return err
		}
		_ = ref
		//
		// fmt.Printf("owner: %v\n", owner)
		// fmt.Printf("repo: %v\n", repo)
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

func localCommits(ctx context.Context, args ...string) ([]*localCommit, error) {
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
	var commits []*localCommit
	for i := 0; i < len(parts)-1; i += 2 {
		hash := strings.TrimSpace(parts[i])
		msg := parts[i+1]

		stackCommitID, err := gitTrailerValue(ctx, msg, commitIDTrailerKey)
		if err != nil {
			return nil, err
		}

		commits = append(commits, &localCommit{
			Hash:          hash,
			Message:       msg,
			StackCommitID: stackCommitID,
		})
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

func commitIDTrailer(commitHash string) string {
	hash := sha1.Sum([]byte(commitHash))
	return fmt.Sprintf("%s: %x", commitIDTrailerKey, hash)
}

func gitTrailerValue(ctx context.Context, msg, trailerKey string) (string, error) {
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

func addStackCommitIDs(ctx context.Context, commits []*localCommit, out io.Writer, baseBranch string) error {
	var rewordCommits []string
	for _, gc := range commits {
		if gc.StackCommitID != "" {
			continue
		}

		gc.StackCommitID = commitIDTrailer(gc.Hash)
		if updateFlags.DryRun {
			fmt.Fprintf(
				out,
				"Reword commit %s to add \"%s\" trailer\n",
				shortHash(gc.Hash),
				gc.StackCommitID,
			)
		} else {
			rewordCommits = append(rewordCommits, gc.Hash)
		}
	}

	if len(rewordCommits) > 0 && !updateFlags.DryRun {
		commitFile, err := os.CreateTemp("", "gh-stack")
		if err != nil {
			return err
		}
		defer commitFile.Close()
		defer os.RemoveAll(commitFile.Name())

		if _, err := commitFile.WriteString(strings.Join(rewordCommits, "\n")); err != nil {
			return err
		}

		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if _, err := git(
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
			return err
		}
	}
	return nil
}
