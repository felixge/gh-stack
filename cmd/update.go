/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
		commits, err := gitLog(cmd.Context(), fmt.Sprintf("%s/%s..", updateFlags.Remote, updateFlags.BaseBranch))
		if err != nil {
			return err
		}
		for _, gc := range commits {
			fmt.Printf("gc: %#v\n", gc)
		}

		remoteURL, err := gitRemoteURL(cmd.Context(), updateFlags.Remote)
		if err != nil {
			return err
		}

		owner, repo, err := parseGitHubRemoteURL(remoteURL)
		if err != nil {
			return err
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		githubConfig, err := loadGitHubConfig(filepath.Join(home, ".config", "gh", "hosts.yml"))
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubConfig.OAuthToken})
		tc := oauth2.NewClient(ctx, ts)
		ghClient := github.NewClient(tc)

		ref, err := getBaseRef(cmd.Context(), ghClient, owner, repo)
		if err != nil {
			return err
		}
		fmt.Printf("ref: %v\n", ref)
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

func git(ctx context.Context, args ...string) (string, error) {
	stdout := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
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

func gitLog(ctx context.Context, args ...string) ([]*gitCommit, error) {
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
	var commits []*gitCommit
	for i := 0; i < len(parts)-1; i += 2 {
		hash := strings.TrimSpace(parts[i])
		msg := strings.TrimSpace(parts[i+1])

		commits = append(commits, &gitCommit{
			Hash:    hash,
			Message: msg,
		})
	}
	return commits, nil
}

type gitCommit struct {
	// Hash is the git commit hash.
	Hash string
	// Message string.
	Message string
}
