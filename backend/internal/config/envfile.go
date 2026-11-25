package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Local configuration is optional. Explicit process values, including empty
// values used to disable integrations, always take precedence.
func loadWithEnvFile(path string, lookup func(string) (string, bool)) (Config, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, errors.New("could not read .env configuration")
	}
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for line := 1; scanner.Scan(); line++ {
			text := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
			if text == "" || strings.HasPrefix(text, "#") {
				continue
			}
			text = strings.TrimPrefix(text, "export ")
			key, value, ok := strings.Cut(text, "=")
			key = strings.TrimSpace(key)
			if !ok || !envKey.MatchString(key) {
				return Config{}, fmt.Errorf("invalid .env assignment on line %d", line)
			}
			value, err = parseEnvValue(strings.TrimSpace(value))
			if err != nil {
				// Never include the line contents: they may contain credentials.
				return Config{}, fmt.Errorf("invalid .env value on line %d", line)
			}
			values[key] = value
		}
		if scanner.Err() != nil {
			return Config{}, errors.New("could not read .env configuration")
		}
	}
	return load(func(key string) (string, bool) {
		if value, ok := lookup(key); ok {
			return value, true
		}
		value, ok := values[key]
		return value, ok
	})
}

func parseEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		for i := 1; i < len(value); i++ {
			if quote == '"' && value[i] == '\\' {
				i++
				continue
			}
			if value[i] != quote {
				continue
			}
			tail := strings.TrimSpace(value[i+1:])
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return "", errors.New("trailing content")
			}
			if quote == '"' {
				return strconv.Unquote(value[:i+1])
			}
			return value[1:i], nil
		}
		return "", errors.New("unclosed quote")
	}
	for i := 1; i < len(value); i++ {
		if value[i] == '#' && (value[i-1] == ' ' || value[i-1] == '\t') {
			return strings.TrimSpace(value[:i]), nil
		}
	}
	return value, nil
}
