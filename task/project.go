package task

import (
	"dwight/util"
	"dwight/util/ignore"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (e *Executor) getProjectStructure() (string, error) {
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

func (e *Executor) printDebugInfo(messages []openai.ChatCompletionMessage, tools []openai.Tool) {
	fmt.Printf("=== DEBUG Task Execution Request ===\n")
	fmt.Printf("Model: %s\n", e.config.Model)

	fmt.Printf("Messages:\n")
	for i, msg := range messages {
		fmt.Printf("  [%d] Role: %s\n", i, msg.Role)
		if msg.Content != "" {
			content := strings.TrimSpace(msg.Content)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			fmt.Printf("      Content: %s\n", content)
		}
		if len(msg.ToolCalls) > 0 {
			fmt.Printf("      Tool Calls (%d):\n", len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				argsStr, _ := json.Marshal(args)
				fmt.Printf("        - %s: %s\n", tc.Function.Name, string(argsStr))
			}
		}
		if msg.ToolCallID != "" {
			fmt.Printf("      Tool Call ID: %s\n", msg.ToolCallID)
		}
	}

	fmt.Printf("Tools (%d):\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s\n", tool.Function.Name)
	}
	fmt.Printf("====================================\n")
}
