package fs

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewMemoryFileSystem(t *testing.T) {
	fs := NewMemoryFileSystem()
	assert.NotNil(t, fs)
	assert.IsType(t, &afero.MemMapFs{}, fs.Fs)
}

func TestNewOsFileSystem(t *testing.T) {
	fs := NewOsFileSystem()
	assert.NotNil(t, fs)
	assert.IsType(t, &afero.OsFs{}, fs.Fs)
}

func TestCreateFile(t *testing.T) {
	fs := NewMemoryFileSystem()
	err := fs.CreateFile("test/file.txt")
	assert.NoError(t, err)

	exists, err := afero.Exists(fs.Fs, "test/file.txt")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestWriteFile(t *testing.T) {
	fs := NewMemoryFileSystem()
	err := fs.WriteFile("test/file.txt", "Hello, World!")
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs.Fs, "test/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(content))
}

func TestIsDir(t *testing.T) {
	fs := NewMemoryFileSystem()
	err := fs.Fs.MkdirAll("test/dir", 0755)
	assert.NoError(t, err)

	isDir := fs.IsDir("test/dir")
	assert.True(t, isDir)

	isDir = fs.IsDir("test/nonexistent")
	assert.False(t, isDir)
}

func TestExecuteFileOperations(t *testing.T) {
	fs := NewMemoryFileSystem()
	operations := []FileOperation{
		{Operation: "CREATE_DIR", Path: "test/dir"},
		{Operation: "CREATE_FILE", Path: "test/dir/file.txt"},
	}

	err := fs.ExecuteFileOperations(operations)
	assert.NoError(t, err)

	isDir := fs.IsDir("test/dir")
	assert.True(t, isDir)

	exists, err := afero.Exists(fs.Fs, "test/dir/file.txt")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestWriteToZip(t *testing.T) {
	fs := NewMemoryFileSystem()
	err := fs.Fs.MkdirAll("test", 0755)
	assert.NoError(t, err)

	err = fs.WriteFile("test/file.txt", "Hello, World!")
	assert.NoError(t, err)

	zipBytes, err := fs.WriteToZip()
	assert.NoError(t, err)
	assert.NotEmpty(t, zipBytes)
}

func TestListFiles(t *testing.T) {
	fs := NewMemoryFileSystem()
	err := fs.Fs.MkdirAll("test", 0755)
	assert.NoError(t, err)

	err = fs.WriteFile("test/file.txt", "Hello, World!")
	assert.NoError(t, err)

	structure, err := fs.ListFiles(".")
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, map[string]interface{}{"test": map[string]interface{}{"file.txt": nil}}, structure)
}
