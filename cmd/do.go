package cmd

import (
	"bufio"
	"context"
	"dwight/api"
	"dwight/config"
	"dwight/prompts"
	"dwight/util"
	"dwight/util/ignore"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

type DoCmd struct {
	inputFile string
	debug     bool
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
	cmd.MarkFlagRequired("input")
	return cmd
}

func (d *DoCmd) run(cmd *cobra.Command, args []string) {
	task, err := os.ReadFile(d.inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading task: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	client := api.NewOpenAIClient(cfg)

	fmt.Println("Executing task...")
	if err := d.executeTask(cfg, client, string(task)); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing task: %v\n", err)
		os.Exit(1)
	}
}

func (d *DoCmd) executeTask(cfg *config.Config, client *api.OpenAIClient, task string) error {
	structure, err := d.getProjectStructure()
	if err != nil {
		return err
	}

	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_file_contents",
				Description: "Get contents of multiple files",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"files": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
					},
					"required": []string{"files"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "modify_file",
				Description: "Modify or create a file with new content",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file to modify or create",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Complete new content for the file",
						},
					},
					"required": []string{"file_path", "content"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_command",
				Description: "Execute a shell command",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "ask_question",
				Description: "Ask the user a clarifying question",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"question": map[string]interface{}{
							"type":        "string",
							"description": "Question to ask the user",
						},
					},
					"required": []string{"question"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "task_complete",
				Description: "Mark the task as completed",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.TaskExecutionSP,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Project structure:\n%s\n\nTask: %s", structure, task),
		},
	}

	for {
		if d.debug {
			fmt.Printf("=== DEBUG Task Execution Request ===\n")
			fmt.Printf("Model: %s\n", cfg.Model)
			fmt.Printf("Messages: %+v\n", messages)
			fmt.Printf("Tools: %+v\n", tools)
			fmt.Printf("====================================\n")
		}

		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    cfg.Model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return err
		}

		if d.debug {
			fmt.Printf("=== DEBUG Task Execution Response ===\n")
			fmt.Printf("Response: %+v\n", resp)
			fmt.Printf("=====================================\n")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		if len(choice.Message.ToolCalls) == 0 {
			if strings.Contains(strings.ToLower(choice.Message.Content), "done") ||
				strings.Contains(strings.ToLower(choice.Message.Content), "complete") ||
				strings.Contains(strings.ToLower(choice.Message.Content), "finished") {
				fmt.Println("Task completed!")
				break
			}
			continue
		}

		taskComplete := false
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Function.Name == "task_complete" {
				taskComplete = true
				break
			}

			if err := d.handleToolCall(toolCall, &messages); err != nil {
				return err
			}
		}

		if taskComplete {
			fmt.Println("Task completed!")
			break
		}
	}

	return nil
}

func (d *DoCmd) getProjectStructure() (string, error) {
	ignorePatterns, err := ignore.LoadPatterns()
	if err != nil {
		return "", err
	}

	var structure strings.Builder
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if util.IsIgnored(path, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			structure.WriteString(path + "/\n")
		} else {
			size := info.Size()
			structure.WriteString(path + " (" + strconv.FormatInt(size, 10) + " bytes)\n")
		}
		return nil
	})
	return structure.String(), err
}

func (d *DoCmd) handleToolCall(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
	switch toolCall.Function.Name {
	case "get_file_contents":
		var args struct {
			Files []string `json:"files"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return err
		}

		ignorePatterns, err := ignore.LoadPatterns()
		if err != nil {
			return err
		}

		contents := make(map[string]string)
		for _, file := range args.Files {
			if util.IsIgnored(file, ignorePatterns) {
				contents[file] = "ERROR: Access to this file is forbidden by ignore patterns"
				continue
			}

			content, err := os.ReadFile(file)
			if err != nil {
				contents[file] = "ERROR: " + err.Error()
			} else {
				contents[file] = string(content)
			}
		}

		contentJSON, err := json.Marshal(contents)
		if err != nil {
			return err
		}

		*messages = append(*messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    string(contentJSON),
			ToolCallID: toolCall.ID,
		})

	case "modify_file":
		var args struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return err
		}

		fmt.Printf("Modifying: %s\n", args.FilePath)

		var oldContent string
		if existing, err := os.ReadFile(args.FilePath); err == nil {
			oldContent = string(existing)
		}
		diff, err := util.UnifiedDiffColored(oldContent, args.Content, args.FilePath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(diff) == "" {
			fmt.Println("No changes detected.")
		} else {
			fmt.Println("Proposed changes:")
			fmt.Println(diff)
		}

		if util.ConfirmAction("Apply these changes?") {
			if err := os.MkdirAll(filepath.Dir(args.FilePath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(args.FilePath, []byte(args.Content), 0644); err != nil {
				return err
			}
			fmt.Printf("Updated %s\n", args.FilePath)
		}

		*messages = append(*messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    "File modification completed",
			ToolCallID: toolCall.ID,
		})

	case "run_command":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return err
		}

		fmt.Printf("Execute: %s\n", args.Command)
		if util.ConfirmAction("Run this command?") {
			cmd := exec.Command("sh", "-c", args.Command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Command failed: %v\n", err)
			}
		}

		*messages = append(*messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    "Command execution completed",
			ToolCallID: toolCall.ID,
		})

	case "ask_question":
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return err
		}

		fmt.Printf("Question: %s\n", args.Question)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Your answer: ")
		answer, _ := reader.ReadString('\n')

		*messages = append(*messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    fmt.Sprintf("Answer: %s", strings.TrimSpace(answer)),
			ToolCallID: toolCall.ID,
		})

	case "task_complete":
		*messages = append(*messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    "Task completion acknowledged",
			ToolCallID: toolCall.ID,
		})
	}

	return nil
}
