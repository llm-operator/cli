package nbtoken

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zchee/go-xdgbasedir"
)

// SaveToken saves the token to a file.
func SaveToken(nbID, token string) error {
	path := tokenFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s: %s\n", nbID, token)
	return err
}

// LoadToken loads the token from a file.
func LoadToken(nbID string) (string, error) {
	file, err := os.Open(tokenFilePath())
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 && parts[0] == nbID {
			return parts[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("key not found")
}

func tokenFilePath() string {
	return filepath.Join(xdgbasedir.ConfigHome(), "llmo", "notebook_tokens.yaml")
}
