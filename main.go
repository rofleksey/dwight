package main

import (
	"dwight/cmd"
	"os"

	"github.com/spf13/cobra"
	"go.szostok.io/version/extension"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dwight",
		Short: "AI-powered work task automation tool",
	}

	rootCmd.AddCommand(cmd.NewFileCmd())
	rootCmd.AddCommand(cmd.NewDoCmd())
	rootCmd.AddCommand(cmd.NewReviewCommand())
	rootCmd.AddCommand(extension.NewVersionCobraCmd(
		extension.WithUpgradeNotice("rofleksey", "dwight"),
	))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
