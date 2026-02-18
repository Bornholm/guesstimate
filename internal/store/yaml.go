package store

import (
	"os"
	"path/filepath"

	"github.com/bornholm/guesstimate/internal/model"
	"gopkg.in/yaml.v3"
)

// YAMLStore handles reading and writing estimation and config files
type YAMLStore struct {
	configFile string
}

// NewYAMLStore creates a new YAML store with the given config file path
func NewYAMLStore(configFile string) *YAMLStore {
	return &YAMLStore{
		configFile: configFile,
	}
}

// DefaultConfigFile returns the default config file name
const DefaultConfigFile = ".guesstimate.yml"

// LoadConfig loads the configuration from the config file
// If no specific config file is set, it searches for the config file
// starting from the current directory and traversing up to parent directories
func (s *YAMLStore) LoadConfig() (*model.Config, error) {
	// If a specific config file is set, use it directly
	if s.configFile != "" {
		return s.loadConfigFromFile(s.configFile)
	}

	// Search for config file starting from current directory and going up
	configPath, err := findConfigFile(DefaultConfigFile)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// No config file found, return default config
		return model.DefaultConfig(), nil
	}

	return s.loadConfigFromFile(configPath)
}

// findConfigFile searches for the config file starting from the current directory
// and traversing up to parent directories until it finds the file or reaches the root
func findConfigFile(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory, file not found
			return "", nil
		}
		dir = parent
	}
}

// loadConfigFromFile loads the configuration from a specific file path
func (s *YAMLStore) loadConfigFromFile(configPath string) (*model.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return model.DefaultConfig(), nil
		}
		return nil, err
	}

	config := &model.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Set category IDs from map keys
	for id, cat := range config.TaskCategories {
		cat.ID = id
		config.TaskCategories[id] = cat
	}

	return config, nil
}

// SaveConfig saves the configuration to the config file
func (s *YAMLStore) SaveConfig(config *model.Config) error {
	// Use configFile if set, otherwise use default
	configPath := s.configFile
	if configPath == "" {
		configPath = DefaultConfigFile
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// LoadEstimation loads an estimation from a file
func (s *YAMLStore) LoadEstimation(path string) (*model.Estimation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	estimation := &model.Estimation{}
	if err := yaml.Unmarshal(data, estimation); err != nil {
		return nil, err
	}

	// Ensure tasks map is initialized
	if estimation.Tasks == nil {
		estimation.Tasks = make(map[model.TaskID]*model.Task)
	}

	// Ensure ordering is initialized
	if estimation.Ordering == nil {
		estimation.Ordering = []model.TaskID{}
	}

	return estimation, nil
}

// LoadOrCreateEstimation loads an estimation from a file, or creates a new one if it doesn't exist
func (s *YAMLStore) LoadOrCreateEstimation(path string, label string) (*model.Estimation, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new estimation
			estimation := model.NewEstimation(label)
			if err := s.SaveEstimation(path, estimation); err != nil {
				return nil, false, err
			}
			return estimation, true, nil
		}
		return nil, false, err
	}

	estimation := &model.Estimation{}
	if err := yaml.Unmarshal(data, estimation); err != nil {
		return nil, false, err
	}

	// Ensure tasks map is initialized
	if estimation.Tasks == nil {
		estimation.Tasks = make(map[model.TaskID]*model.Task)
	}

	// Ensure ordering is initialized
	if estimation.Ordering == nil {
		estimation.Ordering = []model.TaskID{}
	}

	return estimation, false, nil
}

// SaveEstimation saves an estimation to a file
func (s *YAMLStore) SaveEstimation(path string, estimation *model.Estimation) error {
	data, err := yaml.Marshal(estimation)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// CreateEstimation creates a new estimation file
func (s *YAMLStore) CreateEstimation(path string, label string) (*model.Estimation, error) {
	estimation := model.NewEstimation(label)

	if err := s.SaveEstimation(path, estimation); err != nil {
		return nil, err
	}

	return estimation, nil
}

// ListEstimations lists all estimation files in a directory
func (s *YAMLStore) ListEstimations(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yml" {
			// Check if it's an estimation file (ends with .estimation.yml)
			if filepath.Ext(filepath.Base(entry.Name()[:len(entry.Name())-4])) == ".estimation" {
				files = append(files, entry.Name())
			}
		}
	}

	return files, nil
}

// Store interface for dependency injection
type Store interface {
	LoadConfig() (*model.Config, error)
	SaveConfig(config *model.Config) error
	LoadEstimation(path string) (*model.Estimation, error)
	SaveEstimation(path string, estimation *model.Estimation) error
	CreateEstimation(path string, label string) (*model.Estimation, error)
	ListEstimations(dir string) ([]string, error)
}

// Ensure YAMLStore implements Store interface
var _ Store = (*YAMLStore)(nil)
