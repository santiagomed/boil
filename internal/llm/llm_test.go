package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"boil/internal/config"
	"boil/internal/utils"
)

func TestLlm(t *testing.T) {
	cfg, err := config.LoadConfig("")
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}
	llmClient := NewClient(cfg)

	outPath := "tmp/"

	// Create  directory for test output
	err = os.MkdirAll(outPath, 0755)
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	projectDesc := "Create a simple web server in Express that returns 'Hello, World!' when '/' is accessed."

	tests := []struct {
		name     string
		llmFunc  interface{}
		input    interface{}
		filename string
	}{
		{"ProjectDetails", llmClient.GenerateProjectDetails, projectDesc, "project_details.md"},
		{"FileTree", llmClient.GenerateFileTree, "", "file_tree.txt"},
		{"FileOrder", llmClient.DetermineFileOrder, "", "file_order.txt"},
		{"FileOperations", llmClient.GenerateFileOperations, "", "file_operations.json"},
		{"FileContent", llmClient.GenerateFileContent, "", "file_content.txt"},
	}

	var projectDetails, fileTree string
	var fileOrder []string
	fileContent := make(map[string]string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			var err error

			switch tt.name {
			case "ProjectDetails":
				result, err = tt.llmFunc.(func(string) (string, error))(projectDesc)
			case "FileTree":
				result, err = tt.llmFunc.(func(string) (string, error))(projectDetails)
			case "FileOrder":
				result, err = tt.llmFunc.(func(string) ([]string, error))(fileTree)
			case "FileOperations":
				result, err = tt.llmFunc.(func(string, string) ([]utils.FileOperation, error))(projectDetails, fileTree)
			case "FileContent":
				if len(fileOrder) > 0 {
					result, err = tt.llmFunc.(func(string, string, string, map[string]string) (string, error))(
						fileOrder[0], projectDetails, fileTree, fileContent)
				} else {
					t.Fatalf("FileOrder is empty, cannot generate file content")
				}
			}

			if err != nil {
				t.Fatalf("%s error: %v", tt.name, err)
			}

			var output string
			switch v := result.(type) {
			case string:
				output = v
			case []string:
				output = fmt.Sprintf("%v", v)
			case []utils.FileOperation:
				jsonOutput, err := json.MarshalIndent(v, "", "  ")
				if err != nil {
					t.Fatalf("Error marshaling FileOperations: %v", err)
				}
				output = string(jsonOutput)
			default:
				t.Fatalf("Unknown result type for %s", tt.name)
			}

			err = os.WriteFile(filepath.Join(outPath, tt.filename), []byte(output), 0644)
			if err != nil {
				t.Fatalf("Error writing to file: %v", err)
			}

			switch tt.name {
			case "ProjectDetails":
				projectDetails = output
			case "FileTree":
				fileTree = output
			case "FileOrder":
				err = json.Unmarshal([]byte(output), &fileOrder)
				if err != nil {
					t.Fatalf("Error unmarshaling FileOrder: %v", err)
				}
			case "FileContent":
				if len(fileOrder) > 0 {
					fileContent[fileOrder[0]] = output
				}
			}

			t.Logf("Output saved to %s", filepath.Join(outPath, tt.filename))
		})
	}
}
