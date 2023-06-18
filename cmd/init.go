/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"path/filepath"
)

// // initCmd represents the init command
//
//	var initCmd = &cobra.Command{
//		Use:   "init",
//		Short: "A brief description of your command",
//		Long:  ``,
//		RunE:  runInit,
//	}
//
// // TODO: Don't prompt for all these stupid things, just generate a config file
// // with the defaults and let the user update them if needed. Print warnings
// // when certain things can't be determined automatically.
//
//	func runInit(cmd *cobra.Command, _ []string) error {
//		ctx := cmd.Context()
//
//		gitRemoteGuess, err := guessGitRemote(ctx)
//		if err != nil {
//			return err
//		}
//
//		gitRemote, err := prompt("git remote", gitRemoteGuess)
//		if err != nil {
//			return err
//		}
//
//		ownerGuess, repoGuess, err := guessOwnerAndRepo(ctx, gitRemote)
//		if err != nil {
//			return err
//		}
//
//		owner, err := prompt("repository owner", ownerGuess)
//		if err != nil {
//			return err
//		}
//
//		repo, err := prompt("repository name", repoGuess)
//		if err != nil {
//			return err
//		}
//
//		gh, err := initGitHubClient(ctx)
//		if err != nil {
//			// handle no credentials found
//			return err
//		}
//
//		baseBranchGuess, err := guessBaseBranch(ctx, owner, repo, gh)
//		if err != nil {
//			return err
//		}
//
//		baseBranch, err := prompt("base branch", baseBranchGuess)
//		if err != nil {
//			return err
//		}
//
//		config := Config{
//			RepoOwner:         owner,
//			RepoName:          repo,
//			BaseBranch:        baseBranch,
//			GitRemote:         gitRemote,
//			GithubConcurrency: 10,
//		}
//		data, err := yaml.Marshal(config)
//		if err != nil {
//			return err
//		} else if err := os.WriteFile(configFilename, data, 0644); err != nil {
//			return err
//		}
//
//		fmt.Println("\nwrote %s", configFilename)
//
//		return nil
//	}
//
// const configFilename = ".gh-stack.yaml"
//
//	func prompt(question, defaultValue string) (string, error) {
//		reader := bufio.NewReader(os.Stdin)
//		fmt.Printf("%s [%s]: ", question, defaultValue)
//		text, err := reader.ReadString('\n')
//		if err != nil {
//			return "", err
//		}
//		text = strings.TrimSpace(text)
//		if text != "" {
//			return text, nil
//		}
//		return defaultValue, nil
//	}
//
//	func promptInt(question string, defaultValue int) (int, error) {
//		defaultValueStr := strconv.Itoa(defaultValue)
//		inputStr, err := prompt(question, defaultValueStr)
//		if err != nil {
//			return defaultValue, err
//		}
//		return strconv.Atoi(inputStr)
//	}
//
//	func guessGitRemote(ctx context.Context) (string, error) {
//		remotes, err := git(ctx, "remote")
//		if err != nil {
//			return "", err
//		}
//		remote, _, _ := strings.Cut(remotes, "\n")
//		if remote == "" {
//			return "origin", nil
//		}
//		return remote, nil
//	}
//
//	func guessOwnerAndRepo(ctx context.Context, remote string) (string, string, error) {
//		remoteURL, err := gitRemoteURL(ctx, remote)
//		if err != nil {
//			return "", "", err
//		}
//		return parseGitHubRemoteURL(remoteURL)
//	}
func guessOAuthToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	githubConfig, err := loadGitHubConfig(filepath.Join(home, ".config", "gh", "hosts.yml"))
	if err != nil {
		return "", err
	}
	return githubConfig.OAuthToken, nil
}

//
// func guessBaseBranch(ctx context.Context, owner, repo string, gh *github.Client) (string, error) {
// 	ghRepo, _, err := gh.Repositories.Get(ctx, owner, repo)
// 	if err != nil {
// 		return "", err
// 	}
// 	return ghRepo.GetDefaultBranch(), nil
// }
//
// func guessGitRootDir(ctx context.Context) (string, error) {
// 	rootDir, err := git(ctx, "rev-parse", "--show-toplevel")
// 	if err != nil {
// 		return "", err
// 	}
// 	return strings.TrimSpace(rootDir), nil
// }
//
// //	func loadConfig(ctx context.Context) (*Config, error) {
// //		dirs, err := configPaths(ctx)
// //		if err != nil {
// //			return nil, err
// //		}
// //
// //		config := &Config{}
// //		for _, dir := range dirs {
// //			configPath := filepath.Join(dir, configFilename)
// //			data, err := os.ReadFile(configPath)
// //			if os.IsNotExist(err) {
// //				continue
// //			} else if err != nil {
// //				return nil, err
// //			}
// //
// //			if err := yaml.Unmarshal(data, config); err != nil {
// //				return nil, err
// //			}
// //		}
// //
// //		return config, nil
// //	}
// //
// //	func configPaths(ctx context.Context) ([]string, error) {
// //		home, err := os.UserHomeDir()
// //		if err != nil {
// //			return nil, err
// //		}
// //		gitRoot, err := guessGitRootDir(ctx)
// //		if err != nil {
// //			return nil, err
// //		}
// //		return []string{home, gitRoot}, nil
// //	}
// type Config struct {
// 	GitRemote         string `yaml:"git_remote"`
// 	RepoOwner         string `yaml:"repo_owner"`
// 	RepoName          string `yaml:"repo_name"`
// 	BaseBranch        string `yaml:"base_branch"`
// 	GithubConcurrency int    `yaml:"github_concurrency"`
// }
//
// //
// // func (c *Config) applyConfigFiles() error {
// // 	return nil
// // }
//
// func init() {
// 	rootCmd.AddCommand(initCmd)
// }
