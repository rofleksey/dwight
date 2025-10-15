package util

import (
	"strings"

	difflib "github.com/pmezard/go-difflib/difflib"
)

// UnifiedDiffColored returns a colorized unified diff between oldContent and newContent for the given filePath.
// Lines added are green, removed are red, hunk headers are cyan, and file headers are dim gray.
func UnifiedDiffColored(oldContent, newContent, filePath string) (string, error) {
	ud := difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldContent),
		B:        difflib.SplitLines(newContent),
		FromFile: "a/" + filePath,
		ToFile:   "b/" + filePath,
		Context:  3,
	}

	text, err := difflib.GetUnifiedDiffString(ud)
	if err != nil {
		return "", err
	}
	return colorizeDiff(text), nil
}

func colorizeDiff(unified string) string {
	var b strings.Builder
	lines := strings.Split(unified, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			b.WriteString("\x1b[32m") // green
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			b.WriteString("\x1b[31m") // red
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		} else if strings.HasPrefix(line, "@@") {
			b.WriteString("\x1b[36m") // cyan
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		} else if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			b.WriteString("\x1b[90m") // gray
			b.WriteString(line)
			b.WriteString("\x1b[0m")
		} else {
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
