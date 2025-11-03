package godotenv

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Load reads the provided .env files (or ".env" by default) and loads
// key=value pairs into the current process environment. Existing variables
// are preserved.
func Load(filenames ...string) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	var loadErr error
	for _, name := range filenames {
		if err := loadFile(name); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if loadErr == nil {
				loadErr = err
			}
		}
	}
	return loadErr
}

func loadFile(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")

		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if strings.HasPrefix(value, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				value = filepath.Join(home, strings.TrimPrefix(value, "~"))
			}
		}

		_ = os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
