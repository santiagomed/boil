package core

import (
	"os"
	"testing"
	"time"

	"github.com/santiagomed/boil/pkg/config"
	"github.com/santiagomed/boil/pkg/fs"
	"github.com/santiagomed/boil/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLM is a mock implementation of the LLM client
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) GenerateProjectDetails(projectDesc string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDesc)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) GenerateFileTree(projectDetails string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDetails)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) GenerateFileOperations(projectDetails, fileTree string) ([]fs.FileOperation, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDetails, fileTree)
	return args.Get(0).([]fs.FileOperation), args.Error(1)
}

func (m *MockLLM) DetermineFileOrder(fileTree string) ([]string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(fileTree)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockLLM) GenerateFileContent(fileName, projectDetails, fileTree string, previousFiles map[string]string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(fileName, projectDetails, fileTree, previousFiles)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) GenerateReadmeContent(projectDetails string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDetails)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) GenerateGitignoreContent(projectDetails string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDetails)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) GenerateDockerfileContent(projectDetails string) (string, error) {
	time.Sleep(1 * time.Second)
	args := m.Called(projectDetails)
	return args.String(0), args.Error(1)
}

type Publisher struct {
	stepChan chan StepType
	errChan  chan error
}

func NewPublisher() *Publisher {
	return &Publisher{
		stepChan: make(chan StepType),
		errChan:  make(chan error),
	}
}

func (p *Publisher) PublishStep(step StepType) {
	p.stepChan <- step
}

func (p *Publisher) Error(step StepType, err error) {
	p.errChan <- err
}

func TestPipeline_Execute(t *testing.T) {
	mockLLM := new(MockLLM)

	var expectedFileOperations = []fs.FileOperation{
		{Operation: "CREATE_DIR", Path: "src"},
		{Operation: "CREATE_DIR", Path: "src/config"},
		{Operation: "CREATE_DIR", Path: "src/utils"},
		{Operation: "CREATE_DIR", Path: "test"},
		{Operation: "CREATE_FILE", Path: "package.json"},
		{Operation: "CREATE_FILE", Path: "src/index.js"},
		{Operation: "CREATE_FILE", Path: "src/config/config.js"},
		{Operation: "CREATE_FILE", Path: "src/utils/helpers.js"},
		{Operation: "CREATE_FILE", Path: "test/index.test.js"},
		{Operation: "CREATE_FILE", Path: ".env.example"},
	}

	var expectedStructure = map[string]interface{}{
		"src": map[string]interface{}{
			"config": map[string]interface{}{
				"config.js": nil,
			},
			"utils": map[string]interface{}{
				"helpers.js": nil,
			},
			"index.js": nil,
		},
		"test": map[string]interface{}{
			"index.test.js": nil,
		},
		"package.json": nil,
		".env.example": nil,
		".git":         map[string]interface{}{},
		".gitignore":   nil,
		"Dockerfile":   nil,
		"README.md":    nil,
		"file1":        nil,
		"file2":        nil,
	}

	mockLLM.On("GenerateProjectDetails", mock.Anything).Return("Project details", nil)
	mockLLM.On("GenerateFileTree", mock.Anything).Return("File tree", nil)
	mockLLM.On("GenerateFileOperations", mock.Anything, mock.Anything).Return(expectedFileOperations, nil)
	mockLLM.On("DetermineFileOrder", mock.Anything).Return([]string{"file1", "file2"}, nil)
	mockLLM.On("GenerateFileContent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("File content", nil)
	mockLLM.On("GenerateReadmeContent", mock.Anything).Return("README content", nil)
	mockLLM.On("GenerateGitignoreContent", mock.Anything).Return("Gitignore content", nil)
	mockLLM.On("GenerateDockerfileContent", mock.Anything).Return("Dockerfile content", nil)

	cfg := &config.Config{
		ProjectName:  "test-project",
		ModelName:    "test-model",
		OpenAIAPIKey: "test-key",
		GitRepo:      true,
		GitIgnore:    true,
		Dockerfile:   true,
		Readme:       true,
	}

	// Use real FileSystem and StepPublisher
	memFS := fs.NewMemoryFileSystem()
	realPublisher := NewPublisher()

	pipeline := &Pipeline{
		stepManager: NewStepManager(mockLLM, memFS),
		state: &State{
			Config:        cfg,
			PreviousFiles: make(map[string]string),
			Logger:        logger.NewNullLogger(),
		},
		publisher: realPublisher,
	}

	// Create a channel to receive steps
	stepChan := make(chan StepType, 7)
	go func() {
		for step := range realPublisher.stepChan {
			stepChan <- step
		}
	}()

	// Execute the pipeline in a goroutine
	go func() {
		err := pipeline.Execute("Test project description")
		assert.NoError(t, err)
		close(realPublisher.stepChan)
	}()

	// Wait for all steps to complete
	expectedSteps := []StepType{
		GenerateProjectDetails,
		GenerateFileTree,
		GenerateFileOperations,
		ExecuteFileOperations,
		DetermineFileOrder,
		GenerateFileContents,
		CreateOptionalComponents,
		FinalizeProject,
	}
	for _, expectedStep := range expectedSteps {
		select {
		case step := <-stepChan:
			assert.Equal(t, expectedStep, step)
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for step: %v", expectedStep)
		}
	}

	mockLLM.AssertExpectations(t)

	structure, err := memFS.ListFiles()
	assert.NoError(t, err)
	assert.NotEmpty(t, structure)
	assert.Equal(t, expectedStructure, structure)
	_, err = os.Stat("test-project.zip")
	assert.NoError(t, err, "Zip file should exist")
	assert.FileExists(t, "test-project.zip", "Zip file should be created")

	// Clean up the zip file
	os.Remove("test-project.zip")
}
