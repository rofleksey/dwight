package task

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rofleksey/dwight/util"
	"github.com/rofleksey/dwight/util/ignore"
	"github.com/sashabaranov/go-openai"
)

func (e *Executor) handleToolCall(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
	switch toolCall.Function.Name {
	case "get_file_contents":
		return e.handleGetFileContents(toolCall, messages)
	case "modify_files":
		return e.handleModifyFiles(toolCall, messages)
	case "run_command":
		return e.handleRunCommand(toolCall, messages)
	case "ask_question":
		return e.handleAskQuestion(toolCall, messages)
	case "task_complete":
		return e.handleTaskComplete(toolCall, messages)
	default:
		return fmt.Errorf("unknown tool call: %s", toolCall.Function.Name)
	}
}

func (e *Executor) handleGetFileContents(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
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

	fmt.Printf("AI wants to read these files:\n")
	for _, file := range args.Files {
		fmt.Printf("  - %s\n", file)
	}

	contents := make(map[string]string)
	if util.ConfirmAction("Allow reading these files?") {
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
	} else {
		for _, file := range args.Files {
			contents[file] = "ERROR: File reading denied by user"
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
	return nil
}

func (e *Executor) handleModifyFiles(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
	var args struct {
		Files []struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return err
	}

	results := make([]string, 0, len(args.Files))
	for _, file := range args.Files {
		fmt.Printf("Modifying: %s\n", file.FilePath)

		var oldContent string
		if existing, err := os.ReadFile(file.FilePath); err == nil {
			oldContent = string(existing)
		}

		if oldContent != "" && oldContent != file.Content {
			diff, err := util.UnifiedDiffColored(oldContent, file.Content, file.FilePath)
			if err != nil {
				return err
			}

			if strings.TrimSpace(diff) != "" {
				fmt.Println("Proposed changes:")
				fmt.Println(diff)
			}
		} else if oldContent == "" {
			fmt.Println("Creating new file")
		}

		if util.ConfirmAction("Apply these changes?") {
			if err := os.MkdirAll(filepath.Dir(file.FilePath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(file.FilePath, []byte(file.Content), 0644); err != nil {
				return err
			}
			results = append(results, fmt.Sprintf("%s: Updated", file.FilePath))
			fmt.Printf("Updated %s\n", file.FilePath)
		} else {
			results = append(results, fmt.Sprintf("%s: Skipped", file.FilePath))
		}
	}

	*messages = append(*messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    strings.Join(results, "\n"),
		ToolCallID: toolCall.ID,
	})
	return nil
}

func (e *Executor) handleRunCommand(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return err
	}

	fmt.Printf("Execute: %s\n", args.Command)

	resp := map[string]interface{}{
		"command":   args.Command,
		"confirmed": false,
	}

	if util.ConfirmAction("Run this command?") {
		resp["confirmed"] = true

		cmd := exec.Command("sh", "-c", args.Command)

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

		err := cmd.Run()
		exitCode := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exitCode = ee.ExitCode()
			}
			fmt.Printf("Command failed: %v\n", err)
		}

		resp["exit_code"] = exitCode
		resp["stdout"] = stdoutBuf.String()
		resp["stderr"] = stderrBuf.String()
	} else {
		resp["message"] = "Command not executed"
	}

	payload, _ := json.Marshal(resp)

	*messages = append(*messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    string(payload),
		ToolCallID: toolCall.ID,
	})
	return nil
}

func (e *Executor) handleAskQuestion(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
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
	return nil
}

func (e *Executor) handleTaskComplete(toolCall openai.ToolCall, messages *[]openai.ChatCompletionMessage) error {
	*messages = append(*messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    "Task completion acknowledged",
		ToolCallID: toolCall.ID,
	})
	return nil
}
