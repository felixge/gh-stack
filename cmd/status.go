/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// var statusStack stack.StatusStack
		// if err := statusStack.Load(ctx); err != nil {
		// 	return err
		// }
		// table := tablewriter.NewWriter(os.Stdout)
		// table.SetBorder(false)
		// table.SetTablePadding("  ")
		// table.SetNoWhiteSpace(true)
		// for _, commit := range statusStack.LocalStack.Commits {
		// 	status := "new"
		// 	remoteStack := statusStack.RemoteStacks.Stacks[commit.UID]
		// 	if remoteStack != nil {
		// 		status = "unchanged"
		// 	}
		// 	fmt.Fprintf(os.Stdout, "%s r3: r3 ⛈️😴⬇️ %s - <PR URL>\n", status, commit.Oneline())
		// 	table.Append([]string{"", "new", "r1", commit.Oneline()})
		// }
		// // table.Render()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
