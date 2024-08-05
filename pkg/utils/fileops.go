package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

var AppFs afero.Fs

func init() {
	AppFs = afero.NewMemMapFs()
}

// FileOperation represents a single file operation
type FileOperation struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
}

// ExecuteFileOperations performs a series of file operations
func ExecuteFileOperations(operations []FileOperation) error {
	for _, op := range operations {
		if err := ExecuteFileOperation(op); err != nil {
			return fmt.Errorf("error executing operation %s on %s: %w", op.Operation, op.Path, err)
		}
	}
	return nil
}

// ExecuteFileOperation performs a single file operation
func ExecuteFileOperation(op FileOperation) error {
	switch op.Operation {
	case "CREATE_DIR":
		return AppFs.MkdirAll(op.Path, 0755)
	case "CREATE_FILE":
		return CreateFile(op.Path)
	default:
		return fmt.Errorf("unknown operation: %s", op.Operation)
	}
}

// Creates a new file
func CreateFile(path string) error {
	dir := filepath.Dir(path)
	if err := AppFs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	f, err := AppFs.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", path, err)
	}
	defer f.Close()

	return nil
}

// WriteFile creates a new file with the given content or overwrites an existing file with the content
func WriteFile(path string, content string) error {
	return afero.WriteFile(AppFs, path, []byte(content), 0644)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := AppFs.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer sourceFile.Close()

	dstFile, err := AppFs.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return nil
}

// CopyDir recursively copies a directory tree
func CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := AppFs.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	err = AppFs.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := afero.ReadDir(AppFs, src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureDir ensures that the specified directory exists
func EnsureDir(dir string) error {
	return AppFs.MkdirAll(dir, 0755)
}

// MoveDir moves the contents of a directory to another location
func MoveDir(src, dst string) error {
	if err := CopyDir(src, dst); err != nil {
		return fmt.Errorf("error copying directory contents: %w", err)
	}
	if err := AppFs.RemoveAll(src); err != nil {
		return fmt.Errorf("error removing source directory: %w", err)
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := AppFs.Stat(path)
	return !os.IsNotExist(err)
}

// IsDir checks if a path is a directory
func IsDir(path string) bool {
	info, err := AppFs.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// InitializeGitRepo initializes a git repository in the given directory
// Note: This function will need to be handled differently as we can't execute git commands on the in-memory file system
func InitializeGitRepo(dir string) error {
	// For now, we'll just create a .git directory to simulate git initialization
	return AppFs.MkdirAll(filepath.Join(dir, ".git"), 0755)
}

// WriteToZip writes the in-memory file system to a zip file
func WriteToZip(zipPath string) error {
	realFs := afero.NewOsFs()
	zipFile, err := realFs.Create(zipPath)
	if err != nil {
		return fmt.Errorf("error creating zip file: %w", err)
	}
	defer zipFile.Close()

	err = afero.Walk(AppFs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := AppFs.Open(path)
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", path, err)
		}
		defer file.Close()

		// Create zip entry
		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()
		writer, err := zipWriter.Create(path)
		if err != nil {
			return fmt.Errorf("error creating zip entry for %s: %w", path, err)
		}

		// Copy file contents to zip
		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("error writing file %s to zip: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking file system: %w", err)
	}

	return nil
}
