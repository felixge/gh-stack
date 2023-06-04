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
	"os"
	"runtime/trace"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var globalFlags struct {
	Trace   string
	Config  string
	Verbose bool
}

var stopTrace = func() error { return nil }

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-stack",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if globalFlags.Trace != "" {
			file, err := os.Create(globalFlags.Trace)
			if err != nil {
				return err
			} else if err := trace.Start(file); err != nil {
				file.Close()
				return err
			}
			stopTrace = func() error {
				trace.Stop()
				return file.Close()
			}
		}
		return nil
	},
	PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
		return stopTrace()
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&globalFlags.Config, "config", "", "config file (default is $HOME/.gh-stack.yaml)")
	rootCmd.PersistentFlags().StringVar(&globalFlags.Trace, "trace", "", "record a go execution trace to the given file")
	rootCmd.PersistentFlags().BoolVarP(&globalFlags.Verbose, "verbose", "v", false, "output the steps the command is taking")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if globalFlags.Config != "" {
		// Use config file from the flag.
		viper.SetConfigFile(globalFlags.Config)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gh-stack" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gh-stack")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// // If a config file is found, read it in.
	// if err := viper.ReadInConfig(); err == nil {
	// 	fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	// }
}
