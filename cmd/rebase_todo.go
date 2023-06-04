/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rebaseTodoCmd represents the rebaseTodo command
var rebaseTodoCmd = &cobra.Command{
	Use:   "rebase-todo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Hidden: true,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("expected at least 2 args, but got: %d", len(args))
		}
		todoFile := args[len(args)-1]
		oldTodo, err := os.ReadFile(todoFile)
		if err != nil {
			return err
		}
		newTodo, err := rewordCommits(string(oldTodo), args[0:len(args)-1]...)
		if err != nil {
			return err
		}
		return os.WriteFile(todoFile, []byte(newTodo), 0)
	},
}

func init() {
	rootCmd.AddCommand(rebaseTodoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rebaseTodoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rebaseTodoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// rewordCommits changes the git rebase command from "pick" to "reword" for the
// specified commit hashes in a git rebase message. The function accepts a
// rebase message as a string, and a variable number of commit hashes If a
// commit hash is found in the message and its corresponding command is "pick",
// that command is changed to "reword". The function returns the updated rebase
// message. If any of the provided commit hashes are not found in the input
// message, the function returns an error.
func rewordCommits(msg string, commits ...string) (string, error) {
	lines := strings.Split(msg, "\n")

	for i, line := range lines {
		words := strings.Fields(line)
		if len(words) < 2 || words[0] != "pick" {
			continue
		}

		var ok bool
		commits, ok = removeCommit(words[1], commits...)
		if ok {
			lines[i] = strings.Replace(lines[i], "pick", "reword", 1)
		}
	}

	if len(commits) != 0 {
		return "", fmt.Errorf("rebase message did not contain commits: %s", strings.Join(commits, " "))
	}

	return strings.Join(lines, "\n"), nil
}

// removeCommit takes a target commit hash as its first argument and a variadic
// slice of commit hashes as its second argument. It returns a new slice of
// commit hashes with the target commit removed, as well as a boolean
// indicating whether the target commit was found and removed. The function
// compares only the beginning of each commit hash in the list (using
// strings.HasPrefix) to the target commit hash. If the target commit hash is
// not found in the list, 'ok' is false, and the returned slice is the same as
// the input list.
func removeCommit(commit string, commits ...string) (newCommits []string, ok bool) {
	for _, listCommit := range commits {
		if strings.HasPrefix(listCommit, commit) {
			ok = true
		} else {
			newCommits = append(newCommits, listCommit)
		}
	}
	return
}
