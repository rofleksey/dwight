package main

import (
	"dwight/cmd"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{Use: "dwight", Short: "AI-powered work task automation tool"}
	rootCmd.AddCommand(
		cmd.NewDoCmd(),
	)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
