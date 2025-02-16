package shell

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func RunCommand(input string) (string, int, error) {
	cmd := exec.Command("bash", "-c", input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if errStr := stderr.String(); errStr != "" {
		output = errStr
		slog.Warn("command might be failed",
			"command", input,
			"output", output,
		)
	}
	if err == nil {
		return output, 0, nil
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return output, -1, err
	}
	return output, ee.ExitCode(), nil
}

func ExpandHome(input string) (string, error) {
	result := input

	// 1. expand tilda
	if strings.HasPrefix(result, "~/") {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable is not set")
		}
		result = strings.Replace(result, "~/", home+"/", 1)
	} else if result == "~" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable is not set")
		}
		result = home
	}

	// 2. expand env, e.g. $HOME、${HOME}
	for {
		start := strings.Index(result, "$")
		if start == -1 {
			break
		}

		var end int
		var varName string

		if strings.HasPrefix(result[start:], "${") {
			// case of ${VAR} format
			end = strings.Index(result[start:], "}")
			if end == -1 {
				return "", fmt.Errorf("unclosed variable brace in input: %s", input)
			}
			end += start
			varName = result[start+2 : end]
			end++ // go to next of "}"
		} else {
			for i := start + 1; i < len(result); i++ {
				if !isShellVarChar(result[i]) {
					end = i
					break
				}
			}
			if end == 0 {
				end = len(result)
			}
			varName = result[start+1 : end]
		}

		value := os.Getenv(varName)
		if value == "" {
			value = ""
		}

		result = result[:start] + value + result[end:]
	}

	return result, nil
}

func isShellVarChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
