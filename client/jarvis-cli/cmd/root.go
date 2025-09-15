package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jarvis-cli",
	Short: "A CLI client to interact with the Jarvis 2.0 services",
	Long:  `A command-line interface for submitting tasks to the agent system and querying the RAG service.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI: %s", err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Example: rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jarvis-cli.yaml)")
}
