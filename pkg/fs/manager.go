package fs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// FileSystem wraps the Afero Fs interface
type FileSystem struct {
	Fs afero.Fs
}

// NewMemoryFileSystem creates a new in-memory file system
func NewMemoryFileSystem() *FileSystem {
	return &FileSystem{
		Fs: afero.NewMemMapFs(),
	}
}

// NewOsFileSystem creates a new OS-based file system
func NewOsFileSystem() *FileSystem {
	return &FileSystem{
		Fs: afero.NewOsFs(),
	}
}

// FileOperation represents a single file operation
type FileOperation struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
}

// ExecuteFileOperations performs a series of file operations
func (fs *FileSystem) ExecuteFileOperations(operations []FileOperation) error {
	for _, op := range operations {
		if err := fs.ExecuteFileOperation(op); err != nil {
			return fmt.Errorf("error executing operation %s on %s: %w", op.Operation, op.Path, err)
		}
	}
	return nil
}

// ExecuteFileOperation performs a single file operation
func (fs *FileSystem) ExecuteFileOperation(op FileOperation) error {
	switch op.Operation {
	case "CREATE_DIR":
		return fs.Fs.MkdirAll(op.Path, 0755)
	case "CREATE_FILE":
		return fs.CreateFile(op.Path)
	default:
		return fmt.Errorf("unknown operation: %s", op.Operation)
	}
}

// CreateFile creates a new file
func (fs *FileSystem) CreateFile(path string) error {
	dir := filepath.Dir(path)
	if err := fs.Fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	f, err := fs.Fs.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", path, err)
	}
	defer f.Close()

	return nil
}

// WriteFile creates a new file with the given content or overwrites an existing file with the content
func (fs *FileSystem) WriteFile(path string, content string) error {
	err := afero.WriteFile(fs.Fs, path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing file %s: %w", path, err)
	}
	return nil
}

// IsDir checks if a path is a directory
func (fs *FileSystem) IsDir(path string) bool {
	info, err := fs.Fs.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// InitializeGitRepo initializes a git repository in the given directory
// TODO: This function will need to be handled differently as we can't execute git commands on the in-memory file system
func (fs *FileSystem) InitializeGitRepo() error {
	// For now, we'll just create a .git directory to simulate git initialization
	return fs.Fs.MkdirAll(".git", 0755)
}

// WriteToZip writes the in-memory file system to a zip file
func (fs *FileSystem) WriteToZip(zipPath string) error {
	realFs := afero.NewOsFs()
	zipFile, err := realFs.Create(zipPath)
	if err != nil {
		return fmt.Errorf("error creating zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	fileCount := 0
	err = afero.Walk(fs.Fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if path == "." {
			return nil
		}

		// Use path as is, without removing leading slash
		zipPath := path

		if info.IsDir() {
			_, err := zipWriter.Create(zipPath + "/")
			if err != nil {
				return fmt.Errorf("error creating zip entry for directory %s: %w", zipPath, err)
			}
			return nil
		}

		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("error creating zip entry for file %s: %w", zipPath, err)
		}

		file, err := fs.Fs.Open(path)
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", path, err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("error writing file %s to zip: %w", path, err)
		}

		fileCount++
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking file system: %w", err)
	}

	if fileCount == 0 {
		return fmt.Errorf("no files to zip")
	}

	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("error closing zip writer: %w", err)
	}

	return nil
}

// ListFiles lists all files in the filesystem and returns a map representing the directory structure
func (fs *FileSystem) ListFiles() (map[string]interface{}, error) {
	structure := make(map[string]interface{})
	fileCount := 0

	err := afero.Walk(fs.Fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip root directory
		if path == "." {
			return nil
		}

		parts := strings.Split(path, string(os.PathSeparator))
		current := structure
		for i, part := range parts {
			if i == len(parts)-1 {
				if info.IsDir() {
					current[part] = make(map[string]interface{})
				} else {
					current[part] = nil // Use nil to represent files
					fileCount++
				}
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
		return nil
	})

	return structure, err
}
