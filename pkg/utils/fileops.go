package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileOperation represents a single file operation
type FileOperation struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
}

// ExecuteFileOperations performs a series of file operations
func ExecuteFileOperations(baseDir string, operations []FileOperation) error {
	for _, op := range operations {
		if err := ExecuteFileOperation(baseDir, op); err != nil {
			return fmt.Errorf("error executing operation %s on %s: %w", op.Operation, op.Path, err)
		}
	}
	return nil
}

// ExecuteFileOperation performs a single file operation
func ExecuteFileOperation(baseDir string, op FileOperation) error {
	fullPath := filepath.Join(baseDir, op.Path)

	switch op.Operation {
	case "CREATE_DIR":
		return os.MkdirAll(fullPath, 0755)
	case "CREATE_FILE":
		return CreateFile(fullPath)
	default:
		return fmt.Errorf("unknown operation: %s", op.Operation)
	}
}

// Creates a new file
func CreateFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", path, err)
	}
	defer f.Close()

	return nil
}

// Creates a new file with the given content or overwrites an existing file with the content
func WriteFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", path, err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", path, err)
	}

	return nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer sourceFile.Close()

	dstFile, err := os.Create(dst)
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

// CopyDir recursively copies a directory tree, attempting to preserve permissions
func CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
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
			// Skip symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

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
	return os.MkdirAll(dir, os.ModePerm)
}

// CreateCmd creates an *exec.Cmd with the given directory and command
func CreateCmd(dir string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}

// ExecuteCmd executes a command and returns its output
func ExecuteCmd(dir string, name string, args ...string) (string, error) {
	cmd := CreateCmd(dir, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing command '%s %s': %w\nOutput: %s", name, strings.Join(args, " "), err, output)
	}
	return string(output), nil
}

// MoveDir moves the contents of a directory to another location
func MoveDir(src, dst string) error {
	if err := CopyDir(src, dst); err != nil {
		return fmt.Errorf("error copying directory contents: %w", err)
	}
	if err := os.RemoveAll(src); err != nil {
		return fmt.Errorf("error removing source directory: %w", err)
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDir checks if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// InitializeGitRepo initializes a git repository in the given directory
func InitializeGitRepo(dir string) error {
	_, err := ExecuteCmd(dir, "git", "init")
	if err != nil {
		return fmt.Errorf("error initializing git repository: %w", err)
	}
	return nil
}
