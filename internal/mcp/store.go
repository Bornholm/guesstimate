package mcp

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bornholm/guesstimate/internal/model"
	"gopkg.in/yaml.v3"
)

// ChrootedStore is a store that is restricted to a specific directory
type ChrootedStore struct {
	root *os.Root
}

// NewChrootedStore creates a new store restricted to the given directory
func NewChrootedStore(dir string) (*ChrootedStore, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to open root directory: %w", err)
	}

	return &ChrootedStore{
		root: root,
	}, nil
}

// Close closes the root directory
func (s *ChrootedStore) Close() error {
	return s.root.Close()
}

// writeFile writes data to a file within the chrooted directory
func (s *ChrootedStore) writeFile(path string, data []byte) error {
	f, err := s.root.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// LoadEstimation loads an estimation from a file
func (s *ChrootedStore) LoadEstimation(path string) (*model.Estimation, error) {
	data, err := fs.ReadFile(s.root.FS(), path)
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
func (s *ChrootedStore) LoadOrCreateEstimation(path string, label string) (*model.Estimation, bool, error) {
	data, err := fs.ReadFile(s.root.FS(), path)
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
func (s *ChrootedStore) SaveEstimation(path string, estimation *model.Estimation) error {
	data, err := yaml.Marshal(estimation)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := s.root.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return s.writeFile(path, data)
}

// CreateEstimation creates a new estimation file
func (s *ChrootedStore) CreateEstimation(path string, label string) (*model.Estimation, error) {
	estimation := model.NewEstimation(label)

	if err := s.SaveEstimation(path, estimation); err != nil {
		return nil, err
	}

	return estimation, nil
}

// ListEstimations lists all estimation files in a directory
func (s *ChrootedStore) ListEstimations(dir string) ([]string, error) {
	entries, err := fs.ReadDir(s.root.FS(), dir)
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

// DeleteEstimation deletes an estimation file
func (s *ChrootedStore) DeleteEstimation(path string) error {
	return s.root.Remove(path)
}
