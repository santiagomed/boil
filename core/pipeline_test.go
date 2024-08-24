package core

import (
	"context"
	"testing"
	"time"

	"github.com/santiagomed/boil/fs"
	"github.com/santiagomed/boil/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLM is a mock implementation of the LLM client
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) GetCompletion(prompt, responseType string) (string, error) {
	time.Sleep(100 * time.Millisecond)
	args := m.Called(prompt, responseType)
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

	var expectedFileOperations = `
	{
		"operations": [
			{"operation": "CREATE_DIR", "path": "src"},
			{"operation": "CREATE_DIR", "path": "src/config"},
			{"operation": "CREATE_DIR", "path": "src/utils"},
			{"operation": "CREATE_DIR", "path": "test"},
			{"operation": "CREATE_FILE", "path": "package.json"},
			{"operation": "CREATE_FILE", "path": "src/index.js"},
			{"operation": "CREATE_FILE", "path": "src/config/config.js"},
			{"operation": "CREATE_FILE", "path": "src/utils/helpers.js"},
			{"operation": "CREATE_FILE", "path": "test/index.test.js"},
			{"operation": "CREATE_FILE", "path": ".env.example"}
		]
	}`

	var expectedFileList = `
	{
		"files": [
			"src/index.js",
			"src/config/config.js",
			"src/utils/helpers.js",
			"test/index.test.js",
			".env.example",
			"package.json",
			"Dockerfile",
			"README.md",
			".gitignore"
		]
	}`

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
	}

	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "text").Return("Output", nil).Times(13)
	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "json_object").Return(expectedFileOperations, nil).Once()
	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "json_object").Return(expectedFileList, nil).Once()
	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "json_object").Return(`{"package": "json"}`, nil).Once()

	r := &Request{
		ProjectDescription: "description",
		ProjectName:        "test-project",
		GitRepo:            true,
		GitIgnore:          true,
		Dockerfile:         true,
		Readme:             true,
		APIKey:             "test-key",
		ModelName:          "test-model",
	}

	memFS := fs.NewMemoryFileSystem()
	realPublisher := NewPublisher()

	pipeline := &Pipeline{
		stepManager: NewDefaultStepManager(mockLLM, memFS),
		state: &State{
			Request:       r,
			PreviousFiles: make(map[string]string),
			Logger:        logger.NewNullLogger(),
		},
		publisher: realPublisher,
	}

	stepChan := make(chan StepType, 7)
	go func() {
		for step := range realPublisher.stepChan {
			stepChan <- step
		}
	}()

	go func() {
		err := pipeline.Execute(context.Background())
		assert.NoError(t, err)
		close(realPublisher.stepChan)
	}()

	expectedSteps := []StepType{
		GenerateProjectDetails,
		GenerateFileTree,
		GenerateFileOperations,
		ExecuteFileOperations,
		DetermineFileOrder,
		GenerateFileContents,
		CreateOptionalComponents,
		Done,
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

	structure, err := memFS.ListFiles(".")
	assert.NoError(t, err)
	assert.NotEmpty(t, structure)
	assert.Equal(t, expectedStructure, structure)
}

func TestPipeline_Cancel(t *testing.T) {
	mockLLM := new(MockLLM)

	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "text").Return("Project details", nil).Once()
	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "text").Return("File tree", nil).Once()
	mockLLM.On("GetCompletion", mock.AnythingOfType("string"), "json_object").Return(`{"operations": []}`, nil).Once()

	r := &Request{
		ProjectDescription: "Test project description",
		ProjectName:        "test-project",
		GitRepo:            true,
		GitIgnore:          true,
		Dockerfile:         true,
		Readme:             true,
		APIKey:             "test-key",
		ModelName:          "test-model",
	}

	memFS := fs.NewMemoryFileSystem()
	realPublisher := NewPublisher()

	pipeline := &Pipeline{
		stepManager: NewDefaultStepManager(mockLLM, memFS),
		state: &State{
			Request:       r,
			PreviousFiles: make(map[string]string),
			Logger:        logger.NewNullLogger(),
		},
		publisher: realPublisher,
	}

	ctx, cancel := context.WithCancel(context.Background())

	stepChan := make(chan StepType, 8)
	go func() {
		for step := range realPublisher.stepChan {
			stepChan <- step
		}
	}()

	go func() {
		err := pipeline.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		close(realPublisher.stepChan)
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()

	time.Sleep(50 * time.Millisecond)

	completedSteps := []StepType{}
	for {
		select {
		case step := <-stepChan:
			completedSteps = append(completedSteps, step)
		default:
			goto DoneCollecting
		}
	}
DoneCollecting:

	assert.Equal(t, 3, len(completedSteps), "3 steps should have completed")

	for i, step := range completedSteps {
		assert.Equal(t, StepType(i), step, "Steps should be in order")
	}

	mockLLM.AssertNumberOfCalls(t, "GetCompletion", 3)
}
