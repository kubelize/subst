package subst

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubelize/subst/internal/decryptors"
	"github.com/kubelize/subst/internal/decryptors/ejson"
	"github.com/kubelize/subst/internal/kustomize"
	"github.com/kubelize/subst/internal/utils"
	"github.com/kubelize/subst/internal/wrapper"
	"github.com/kubelize/subst/pkg/config"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Subst represents a simplified subst processor for CMP
type Subst struct {
	Kustomization  *kustomize.Kustomize
	Manifests      [][]byte // Store as byte slices for simplicity
	Substitutions  map[string]interface{}
	EjsonDecryptor *ejson.EjsonDecryptor // Add ejson decryptor
	Config         config.Configuration  // Store full config for ejson keys
}

// NewSubst creates a new simplified Subst instance
func NewSubst(config config.Configuration) (*Subst, error) {
	k, err := kustomize.NewKustomize(config.RootDirectory, config.KustomizeBuildOptions)
	if err != nil {
		return nil, err
	}

	// Get environment variables that match the regex
	envVars, err := GetVariables(config.EnvRegex)
	if err != nil {
		return nil, err
	}

	// Initialize ejson decryptor with standard key paths
	// Prefer /opt/ejson/keys (for containers), fall back to ~/.ejson/keys
	keyDir := ""
	if _, err := os.Stat("/opt/ejson/keys"); err == nil {
		keyDir = "/opt/ejson/keys"
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		userKeyDir := filepath.Join(homeDir, ".ejson", "keys")
		if _, err := os.Stat(userKeyDir); err == nil {
			keyDir = userKeyDir
		}
	}
	log.Debug().Msgf("Using ejson key directory: %s", keyDir)

	ejsonDecryptor, err := ejson.NewEJSONDecryptor(
		decryptors.DecryptorConfig{SkipDecrypt: config.SkipDecrypt},
		keyDir,
		config.EjsonKey..., // Pass ejson keys from config
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ejson decryptor: %w", err)
	}

	subst := &Subst{
		Kustomization:  k,
		Manifests:      [][]byte{},
		Substitutions:  envVars,
		EjsonDecryptor: ejsonDecryptor,
		Config:         config,
	}

	// Load subst.yaml files from kustomize paths
	err = subst.loadSubstFiles()
	if err != nil {
		log.Warn().Msgf("Failed to load subst files: %v", err)
	}

	// Always try to load ejson files (will use keys from disk if no explicit keys provided)
	err = subst.loadEjsonFiles()
	if err != nil {
		log.Warn().Msgf("Failed to load ejson files: %v", err)
	}

	return subst, nil
}

// loadSubstFiles loads subst.yaml files following kustomization resource structure
// Loads from: ancestors (parents), root, and resource directories from kustomization.yaml
// This ensures only active overlays are loaded (respecting commented resources)
func (s *Subst) loadSubstFiles() error {
	// Collect all relevant paths in order
	pathsToLoad := make(map[string]bool)
	
	// 1. Add ancestor paths (going up from root)
	currentPath := s.Kustomization.Root
	for {
		pathsToLoad[currentPath] = true
		parentPath := filepath.Dir(currentPath)
		
		// Stop at filesystem root or when we can't go higher
		if parentPath == currentPath || parentPath == "/" || parentPath == "." {
			break
		}
		currentPath = parentPath
	}
	
	// 2. Add resource paths from kustomization.yaml (only active, not commented)
	for _, resourcePath := range s.Kustomization.GetResourcePaths() {
		absPath, err := filepath.Abs(resourcePath)
		if err == nil {
			pathsToLoad[absPath] = true
		}
	}
	
	// Convert to sorted list (shallowest first for proper override order)
	var paths []string
	for path := range pathsToLoad {
		paths = append(paths, path)
	}
	
	// Sort by depth - shallower paths first (fewer separators)
	// This ensures parent configs are loaded before child configs
	sort.Slice(paths, func(i, j int) bool {
		return strings.Count(paths[i], string(filepath.Separator)) < 
		       strings.Count(paths[j], string(filepath.Separator))
	})
	
	// Load subst.yaml files in order (parents first, then children)
	// Deep merge ensures child values override parent values
	for _, path := range paths {
		err := s.loadSubstFromPath(path)
		if err != nil {
			log.Debug().Msgf("No subst file in %s: %v", path, err)
		}
	}

	return nil
}

// loadSubstFromPath loads subst.yaml files from a specific path
func (s *Subst) loadSubstFromPath(basePath string) error {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if entry.Name() == "subst.yaml" {
			filePath := filepath.Join(basePath, entry.Name())
			log.Debug().Msgf("Loading subst file: %s", filePath)
			substData, err := s.loadSubstFile(filePath)
			if err != nil {
				log.Warn().Msgf("Failed to load %s: %v", filePath, err)
				continue
			}

			// Deep merge subst data into substitutions
			s.Substitutions = utils.DeepMerge(s.Substitutions, substData)
		}
	}

	return nil
}

// loadSubstFile loads a single subst.yaml file and returns the entire structure
func (s *Subst) loadSubstFile(filePath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var substFile map[string]interface{}
	err = yaml.Unmarshal(content, &substFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}

	// Return the entire file structure
	return substFile, nil
}

// loadEjsonFiles finds and decrypts ejson files, making their values available for substitution
func (s *Subst) loadEjsonFiles() error {
	if s.EjsonDecryptor == nil {
		return nil
	}

	// Find all .ejson files in the kustomize directory tree
	ejsonFiles, err := s.findEjsonFiles()
	if err != nil {
		return fmt.Errorf("failed to find ejson files: %w", err)
	}

	for _, ejsonFile := range ejsonFiles {
		log.Debug().Msgf("Processing ejson file for substitution: %s", ejsonFile)

		// Read the ejson file
		content, err := os.ReadFile(ejsonFile)
		if err != nil {
			log.Warn().Msgf("Failed to read ejson file %s: %v", ejsonFile, err)
			continue
		}

		// Check if it's encrypted
		isEncrypted, err := s.EjsonDecryptor.IsEncrypted(content)
		if err != nil || !isEncrypted {
			log.Debug().Msgf("File %s is not encrypted ejson, skipping", ejsonFile)
			continue
		}

		// Decrypt the file
		decryptedData, err := s.EjsonDecryptor.Decrypt(content)
		if err != nil {
			log.Warn().Msgf("Failed to decrypt ejson file %s: %v", ejsonFile, err)
			continue
		}
		log.Debug().Msgf("Successfully decrypted ejson file %s with %d fields", ejsonFile, len(decryptedData))

		// Add the decrypted data under the ejson namespace to avoid conflicts
		if ejsonData, exists := s.Substitutions["ejson"]; exists {
			if ejsonMap, ok := ejsonData.(map[string]interface{}); ok {
				// Merge into existing ejson namespace
				for key, value := range decryptedData {
					// Skip the _public_key field
					if key != "_public_key" {
						ejsonMap[key] = value
					}
				}
			}
		} else {
			// Create new ejson namespace
			ejsonNamespace := make(map[string]interface{})
			for key, value := range decryptedData {
				// Skip the _public_key field
				if key != "_public_key" {
					ejsonNamespace[key] = value
				}
			}
			s.Substitutions["ejson"] = ejsonNamespace
		}
		log.Debug().Msgf("Successfully loaded ejson file %s under .ejson namespace", ejsonFile)
	}

	return nil
}

// findEjsonFiles finds all .ejson files in the current directory and subdirectories
func (s *Subst) findEjsonFiles() ([]string, error) {
	var ejsonFiles []string

	err := filepath.Walk(s.Config.RootDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".ejson") {
			ejsonFiles = append(ejsonFiles, path)
		}

		return nil
	})

	return ejsonFiles, err
}




// Build processes kustomize output with gomplate templates
func (s *Subst) Build() error {
	if s.Kustomization == nil {
		return fmt.Errorf("no kustomization configured")
	}

	log.Debug().Msg("Building resources with simplified approach")

	// Get YAML output from kustomize
	yamlContent := s.Kustomization.GetYAML()
	if yamlContent == "" {
		return fmt.Errorf("kustomize produced no output")
	}

	// Use all substitution data directly for gomplate processing
	log.Debug().Msgf("Template data: %+v", s.Substitutions)

	// Process the entire YAML content with gomplate
	processedYAMLBytes, err := wrapper.ProcessGomplateTemplate([]byte(yamlContent), s.Substitutions, "")
	if err != nil {
		return fmt.Errorf("failed to process with gomplate: %w", err)
	}

	s.Manifests = [][]byte{processedYAMLBytes}

	log.Debug().Msgf("Built %d manifest(s)", len(s.Manifests))
	return nil
}
