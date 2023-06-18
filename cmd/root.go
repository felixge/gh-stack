/*
Copyright © 2023 Felix Geisendörfer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var rootFlags struct {
		Trace   string
		Config  string
		Verbose bool
	}

	var stopTrace = func() error { return nil }

	var rootCmd = &cobra.Command{
		Use:   "gh-stack",
		Short: "A brief description of your application",
		Long:  ``,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if rootFlags.Trace != "" {
				if err := pprof.StartCPUProfile(io.Discard); err != nil {
					return err
				}
				file, err := os.Create(rootFlags.Trace)
				if err != nil {
					return err
				} else if err := trace.Start(file); err != nil {
					file.Close()
					return err
				}

				ctx, task := trace.NewTask(cmd.Context(), cmd.Name())
				cmd.SetContext(ctx)

				stopTrace = func() error {
					task.End()
					trace.Stop()
					pprof.StopCPUProfile()
					return file.Close()
				}
			}
			return nil
		},
		PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
			return stopTrace()
		},
	}
	rootCmd.PersistentFlags().StringVar(&rootFlags.Config, "config", "", "config file")
	rootCmd.PersistentFlags().StringVar(&rootFlags.Trace, "trace", "", "record a go execution trace to the given file")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.Verbose, "verbose", "v", false, "verbose output")

	// rebaseAddTrailerCmd represents the rebaseAddTrailer command
	var rebaseAddTrailerCmd = &cobra.Command{
		Use:    "rebase-add-trailer",
		Hidden: true,
		RunE:   runRebaseAddTrailer,
	}
	rootCmd.AddCommand(rebaseAddTrailerCmd)

	// rebaseEditTodoCmd represents the rebaseEditTodo command
	var rebaseEditTodoCmd = &cobra.Command{
		Use:    "rebase-edit-todo",
		Hidden: true,
		RunE:   runRebaseEditTodo,
	}
	rootCmd.AddCommand(rebaseEditTodoCmd)

	var pushFlags pushFlags
	var pushCmd = &cobra.Command{
		Use:          "push",
		Short:        "Pushes the local commit stack and creates/updates PRs for it.",
		Long:         ``,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pushFlags.Verbose = rootFlags.Verbose
			return runPush(cmd, args, pushFlags)
		},
	}
	pushCmd.Flags().StringVarP(&pushFlags.Remote, "remote", "r", "origin", "git remote to interact with")
	pushCmd.Flags().BoolVarP(&pushFlags.DryRun, "dry-run", "n", false, "show steps without executing them")
	pushCmd.Flags().StringVarP(&pushFlags.Base, "base", "b", "main", "base branch to target with pull requests")
	rootCmd.AddCommand(pushCmd)

	cobra.OnInitialize(func() {
		if rootFlags.Config != "" {
			viper.SetConfigFile(rootFlags.Config)
			cobra.CheckErr(viper.ReadInConfig())
			if rootFlags.Verbose {
				rootCmd.Printf("loaded config %q\n", rootFlags.Config)
			}
		} else {
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)
			wd, err := os.Getwd()
			cobra.CheckErr(err)
			for i, dir := range []string{home, wd} {
				configFile := filepath.Join(dir, ".gh-stack.yaml")
				viper.SetConfigFile(configFile)
				var err error
				if i == 0 {
					err = viper.ReadInConfig()
				} else {
					err = viper.MergeInConfig()
				}
				if err == nil && rootFlags.Verbose {
					rootCmd.Printf("loaded config %q\n", configFile)
				}
				if err != nil && !os.IsNotExist(err) {
					cobra.CheckErr(err)
				}
			}
		}
		fixViperQuirk(pushCmd)
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func fixViperQuirk(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			viper.BindPFlag(f.Name, f)
			f.Value.Set(viper.GetString(f.Name))
		})
	}
}
