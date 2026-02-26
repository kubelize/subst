package kustomize

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Kustomize struct {
	Root             string
	Paths            []string
	ResourcePaths    []string // Paths from kustomization.yaml resources field
	BuildYAML        string
	BuildOptions     string
}

// KustomizationFile represents a simplified kustomization.yaml structure
type KustomizationFile struct {
	Resources []string `yaml:"resources"`
}

func NewKustomize(root string, buildOptions string) (*Kustomize, error) {
	k := &Kustomize{Root: root, BuildOptions: buildOptions}
	if err := k.build(); err != nil {
		return nil, err
	}
	if err := k.findKustomizationFiles(); err != nil {
		return nil, err
	}
	if err := k.parseKustomizationResources(); err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Kustomize) build() error {
	// Use kustomize binary
	args := []string{"build", k.Root}

	// Add build options if provided
	if k.BuildOptions != "" {
		// Split build options by spaces and add to args
		options := strings.Fields(k.BuildOptions)
		args = append(args, options...)
	}

	cmd := exec.Command("kustomize", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w, stderr: %s", err, stderr.String())
	}

	k.BuildYAML = stdout.String()
	return nil
}

func (k *Kustomize) findKustomizationFiles() error {
	// Find all kustomization files in the directory tree
	err := filepath.WalkDir(k.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filename := strings.ToLower(d.Name())
		if filename == "kustomization.yaml" || filename == "kustomization.yml" || filename == "kustomization" {
			k.Paths = append(k.Paths, filepath.Dir(path))
		}

		return nil
	})

	return err
}

func (k *Kustomize) GetYAML() string {
	return k.BuildYAML
}


func (k *Kustomize) GetResourcePaths() []string {
	return k.ResourcePaths
}

// parseKustomizationResources parses the kustomization.yaml in Root and extracts resource paths
func (k *Kustomize) parseKustomizationResources() error {
	// Find the kustomization file
	var kustomizationPath string
	for _, name := range []string{"kustomization.yaml", "kustomization.yml", "Kustomization"} {
		path := filepath.Join(k.Root, name)
		if _, err := os.Stat(path); err == nil {
			kustomizationPath = path
			break
		}
	}
	
	if kustomizationPath == "" {
		// No kustomization file found, that's ok
		return nil
	}
	
	// Read and parse the file
	content, err := os.ReadFile(kustomizationPath)
	if err != nil {
		return err
	}
	
	var kustFile KustomizationFile
	if err := yaml.Unmarshal(content, &kustFile); err != nil {
		return fmt.Errorf("failed to parse kustomization file: %w", err)
	}
	
	// Resolve resource paths relative to Root
	for _, resource := range kustFile.Resources {
		// Clean the path and resolve it relative to Root
		resourcePath := filepath.Clean(filepath.Join(k.Root, resource))
		
		// Check if it's a directory
		info, err := os.Stat(resourcePath)
		if err == nil && info.IsDir() {
			k.ResourcePaths = append(k.ResourcePaths, resourcePath)
		}
	}
	
	return nil
}
func (k *Kustomize) GetPaths() []string {
	return k.Paths
}
