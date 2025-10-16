package task

import (
	"dwight/util"
	"dwight/util/ignore"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

			if strings.HasSuffix(path, ".go") {
				content, err := os.ReadFile(path)
				if err == nil {
					lines := strings.Split(string(content), "\n")
					var snippet strings.Builder
					for i, line := range lines {
						if i >= e.cfg.SnippetMaxLines {
							snippet.WriteString("// ... (truncated)\n")
							break
						}
						snippet.WriteString(line + "\n")
					}
					structure.WriteString("```go\n")
					structure.WriteString(snippet.String())
					structure.WriteString("```\n")
				}
			}
		}
		return nil
	})
	return structure.String(), err
}
