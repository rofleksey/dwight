package cmd

import (
	"fmt"
	"os"

	"github.com/rofleksey/dwight/api"
	"github.com/rofleksey/dwight/config"
	"github.com/rofleksey/dwight/task"
	"github.com/rofleksey/dwight/util"
	"github.com/spf13/cobra"
)

type FileCmd struct {
	inputFile string
	yes       bool
}

func NewFileCmd() *cobra.Command {
	fileCmd := &FileCmd{}
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Execute task from file",
		Run:   fileCmd.run,
	}
	cmd.Flags().StringVarP(&fileCmd.inputFile, "input", "i", "", "Task description file")
	cmd.Flags().BoolVarP(&fileCmd.yes, "yes", "y", false, "Automatically answer Yes to all confirmations (except ask_question)")
	cmd.MarkFlagRequired("input")
	return cmd
}

func (d *FileCmd) run(_ *cobra.Command, _ []string) {
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
