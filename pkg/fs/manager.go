package fs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/santiagomed/boil/pkg/utils"
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
	log := utils.GetLogger()
	err := afero.WriteFile(fs.Fs, path, []byte(content), 0644)
	if err != nil {
		log.Error().Msgf("Error writing file %s: %v", path, err)
		return fmt.Errorf("error writing file %s: %w", path, err)
	}
	log.Debug().Msgf("Successfully wrote file: %s", path)
	return nil
}

// CopyFile copies a file from src to dst
func (fs *FileSystem) CopyFile(src, dst string) error {
	sourceFile, err := fs.Fs.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer sourceFile.Close()

	dstFile, err := fs.Fs.Create(dst)
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
func (fs *FileSystem) CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := fs.Fs.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	err = fs.Fs.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := afero.ReadDir(fs.Fs, src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = fs.CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = fs.CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureDir ensures that the specified directory exists
func (fs *FileSystem) EnsureDir(dir string) error {
	return fs.Fs.MkdirAll(dir, 0755)
}

// MoveDir moves the contents of a directory to another location
func (fs *FileSystem) MoveDir(src, dst string) error {
	if err := fs.CopyDir(src, dst); err != nil {
		return fmt.Errorf("error copying directory contents: %w", err)
	}
	if err := fs.Fs.RemoveAll(src); err != nil {
		return fmt.Errorf("error removing source directory: %w", err)
	}
	return nil
}

// FileExists checks if a file exists
func (fs *FileSystem) FileExists(path string) bool {
	_, err := fs.Fs.Stat(path)
	return err == nil
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
	log := utils.GetLogger()
	log.Debug().Msgf("Starting to write zip file: %s", zipPath)

	// List all files before zipping
	if err := fs.ListFiles(); err != nil {
		log.Error().Msgf("Error listing files before zip creation: %v", err)
	}

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
			log.Error().Msgf("Error walking path %s: %v", path, err)
			return err
		}

		// Skip root directory
		if path == "." {
			return nil
		}

		// Use path as is, without removing leading slash
		zipPath := path

		if info.IsDir() {
			log.Debug().Msgf("Adding directory to zip: %s", zipPath)
			_, err := zipWriter.Create(zipPath + "/")
			if err != nil {
				log.Error().Msgf("Error creating zip entry for directory %s: %v", zipPath, err)
				return fmt.Errorf("error creating zip entry for directory %s: %w", zipPath, err)
			}
			return nil
		}

		log.Debug().Msgf("Adding file to zip: %s", zipPath)
		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			log.Error().Msgf("Error creating zip entry for file %s: %v", zipPath, err)
			return fmt.Errorf("error creating zip entry for file %s: %w", zipPath, err)
		}

		file, err := fs.Fs.Open(path)
		if err != nil {
			log.Error().Msgf("Error opening file %s: %v", path, err)
			return fmt.Errorf("error opening file %s: %w", path, err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			log.Error().Msgf("Error writing file %s to zip: %v", path, err)
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

	log.Debug().Msgf("Successfully added %d files to zip", fileCount)

	err = zipWriter.Close()
	if err != nil {
		log.Error().Msgf("Error closing zip writer: %v", err)
		return fmt.Errorf("error closing zip writer: %w", err)
	}

	log.Debug().Msgf("Successfully created zip file: %s", zipPath)
	return nil
}

// ListFiles lists all files in the filesystem
func (fs *FileSystem) ListFiles() error {
	log := utils.GetLogger()
	fileCount := 0
	err := afero.Walk(fs.Fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msgf("Error walking path %s: %v", path, err)
			return err
		}
		// Skip root directory
		if path == "." {
			return nil
		}

		if info.IsDir() {
			log.Debug().Msgf("Directory: %s", path)
		} else {
			fileCount++
			log.Debug().Msgf("File: %s", path)
		}
		return nil
	})
	log.Debug().Msgf("Total files found: %d", fileCount)
	return err
}
