package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var autoConfirm bool

func SetAutoConfirm(v bool) { autoConfirm = v }

func IsIgnored(file string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, file)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func ConfirmAction(prompt string) bool {
	if autoConfirm {
		fmt.Printf("%s (y/N): y\n", prompt)
		return true
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/N): ", prompt)
	response, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}
