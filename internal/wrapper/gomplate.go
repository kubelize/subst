package wrapper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/yaml.v3"
)

// ProcessGomplateTemplate processes templates using the gomplate binary.
// Gomplate must be installed and available in PATH.
func ProcessGomplateTemplate(templateContent []byte, envData map[string]interface{}, envRegex string) ([]byte, error) {
	// Verify gomplate is available
	if _, err := exec.LookPath("gomplate"); err != nil {
		return nil, fmt.Errorf("gomplate binary not found in PATH. Please install from: https://github.com/hairyhenderson/gomplate#installation")
	}

	return processWithGomplateBinary(templateContent, envData)
}

// processWithGomplateBinary uses the gomplate command-line tool
func processWithGomplateBinary(templateContent []byte, envData map[string]interface{}) ([]byte, error) {
	// Create a temporary file with the context data
	contextData, err := yaml.Marshal(envData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Create temp files for context data
	tmpFile, err := os.CreateTemp("", "subst-context-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = tmpFile.Write(contextData)
	if err != nil {
		return nil, fmt.Errorf("failed to write context data: %w", err)
	}
	tmpFile.Close()

	// Use gomplate with context file
	cmd := exec.Command("gomplate", "--context", fmt.Sprintf(".=%s", tmpFile.Name()))

	// Set basic environment for gomplate (keeping existing env vars)
	cmd.Env = os.Environ()

	// Pipe template content to gomplate
	cmd.Stdin = bytes.NewReader(templateContent)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute gomplate
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gomplate execution failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
