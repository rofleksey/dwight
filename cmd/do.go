package cmd

import (
	"dwight/api"
	"dwight/config"
	"dwight/task"
	"dwight/util"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type DoCmd struct {
	inputFile string
	debug     bool
	yes       bool
}

func NewDoCmd() *cobra.Command {
	doCmd := &DoCmd{}
	cmd := &cobra.Command{
		Use:   "do",
		Short: "Execute task",
		Run:   doCmd.run,
	}
	cmd.Flags().StringVarP(&doCmd.inputFile, "input", "i", "", "Task description file")
	cmd.Flags().BoolVarP(&doCmd.debug, "debug", "d", false, "Enable debug mode")
	cmd.Flags().BoolVarP(&doCmd.yes, "yes", "y", false, "Automatically answer Yes to all confirmations (except ask_question)")
	cmd.MarkFlagRequired("input")
	return cmd
}

func (d *DoCmd) run(_ *cobra.Command, _ []string) {
	taskContent, err := os.ReadFile(d.inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading task: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	util.SetAutoConfirm(d.yes)

	client := api.NewOpenAIClient(cfg)
	executor := task.NewExecutor(client, cfg)

	fmt.Println("Executing task...")
	if err := executor.Execute(string(taskContent)); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing task: %v\n", err)
		os.Exit(1)
	}
}
