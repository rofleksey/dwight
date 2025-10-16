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

type ReviewCommand struct {
	yes bool
}

func NewReviewCommand() *cobra.Command {
	reviewCommand := &ReviewCommand{}
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Handle review comments",
		Run:   reviewCommand.run,
	}
	cmd.Flags().BoolVarP(&reviewCommand.yes, "yes", "y", false, "Automatically answer Yes to all confirmations")
	return cmd
}

func (b *ReviewCommand) run(_ *cobra.Command, _ []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.BitbucketToken == "" {
		fmt.Fprintf(os.Stderr, "Bitbucket token not configured.\n")
		os.Exit(1)
	}

	util.SetAutoConfirm(b.yes)

	client := api.NewOpenAIClient(cfg)
	executor := task.NewExecutor(client, cfg)

	comments, err := executor.GetBitbucketReviewComments(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching review comments: %v\n", err)
		os.Exit(1)
	}

	task := fmt.Sprintf("Implement the requested changes from the following Bitbucket pull request review comments:\n\n%s\n\nFocus on addressing specific code changes mentioned in the comments. Provide complete file contents when making changes.", comments)

	fmt.Println("Processing review comments...")
	if err := executor.Execute(task); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing review comments: %v\n", err)
		os.Exit(1)
	}
}
