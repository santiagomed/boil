package tempdir

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/santiagomed/boil/pkg/config"
	"github.com/santiagomed/boil/pkg/utils"
)

// Manager handles the creation and management of temporary directories
type Manager struct {
	config     *config.Config
	tempDirs   map[string]string
	mutex      sync.Mutex
	cleanupRun bool
}

// NewManager creates a new temporary directory manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		tempDirs: make(map[string]string),
	}
}

// CreateTempDir creates a new temporary directory and returns its path
func (m *Manager) CreateTempDir(prefix string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	tempDir := filepath.Join("/", prefix)
	err := utils.AppFs.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	m.tempDirs[prefix] = tempDir
	return tempDir, nil
}

// GetTempDir returns the path of an existing temporary directory
func (m *Manager) GetTempDir(prefix string) (string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	dir, exists := m.tempDirs[prefix]
	return dir, exists
}

// RemoveTempDir removes a specific temporary directory
func (m *Manager) RemoveTempDir(prefix string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	dir, exists := m.tempDirs[prefix]
	if !exists {
		return fmt.Errorf("temporary directory with prefix %s does not exist", prefix)
	}

	err := utils.AppFs.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("failed to remove temporary directory %s: %w", dir, err)
	}

	delete(m.tempDirs, prefix)
	return nil
}

// Cleanup removes all temporary directories created by this manager
func (m *Manager) Cleanup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.cleanupRun {
		return nil // Cleanup has already been run
	}

	var errors []error
	for prefix, dir := range m.tempDirs {
		err := utils.AppFs.RemoveAll(dir)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to remove temporary directory %s: %w", dir, err))
		} else {
			delete(m.tempDirs, prefix)
		}
	}

	m.cleanupRun = true

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred during cleanup: %v", errors)
	}
	return nil
}
