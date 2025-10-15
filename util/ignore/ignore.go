package ignore

import (
	"bufio"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
)

const ignoreFilename = ".dwightignore"

var defaultPatterns = []string{
	".idea/**",
	".*ignore",
	".git/**",
	"LICENSE",
	"go.sum",
}

func LoadPatterns() ([]string, error) {
	var patterns []string

	patterns = append(patterns, defaultPatterns...)

	home, err := os.UserHomeDir()
	if err == nil {
		homeIgnore, err := readIgnoreFile(filepath.Join(home, ignoreFilename))
		if err == nil {
			patterns = append(patterns, homeIgnore...)
		}
	}

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == ignoreFilename {
			dir := filepath.Dir(path)
			filePatterns, err := readIgnoreFile(path)
			if err == nil {
				for _, pattern := range filePatterns {
					if dir != "." {
						patterns = append(patterns, filepath.Join(dir, pattern))
					} else {
						patterns = append(patterns, pattern)
					}
				}
			}
		}
		return nil
	})

	return patterns, err
}

func readIgnoreFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, scanner.Err()
}
