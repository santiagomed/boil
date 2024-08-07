package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santiagomed/boil/pkg/fs"
)

type Cache struct {
	outPath string
	t       *testing.T
}

func NewCache(outPath string, t *testing.T) *Cache {
	return &Cache{
		outPath: outPath,
		t:       t,
	}
}

func (c *Cache) Get(filename string) (string, error) {
	path := filepath.Join(c.outPath, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cache miss: %v", err)
	}
	c.t.Logf("Cache hit: %s", path)
	return string(content), nil
}

func (c *Cache) Set(filename string, content string) error {
	path := filepath.Join(c.outPath, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing cache: %v", err)
	}
	c.t.Logf("Cache set: %s", path)
	return nil
}

func TestLlmSequential(t *testing.T) {
	t.Log("Starting TestLlmSequential")

	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY is not set, skipping test")
	}

	cfg := LlmConfig{
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		ModelName:    "gpt-4o-mini",
		ProjectName:  "test-project",
	}
	llmClient, err := NewClient(&cfg)
	if err != nil {
		t.Fatalf("Error creating LLM client: %v", err)
	}
	t.Log("LLM client initialized")

	outPath := "tmp/"
	err = os.MkdirAll(outPath, 0755)
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	t.Logf("Created output directory: %s", outPath)

	cache := NewCache(outPath, t)

	projectDesc := "Create a simple web server in Express that returns 'Hello, World!' when '/' is accessed."
	t.Logf("Project description: %s", projectDesc)

	// Step 1: Generate Project Details
	t.Log("Step 1: Generating Project Details")
	projectDetails, err := cache.Get("project_details.md")
	if err != nil {
		projectDetails, err = llmClient.GenerateProjectDetails(projectDesc)
		if err != nil {
			t.Fatalf("ProjectDetails error: %v", err)
		}
		if err := cache.Set("project_details.md", projectDetails); err != nil {
			t.Logf("Failed to cache project details: %v", err)
		}
	}
	t.Log("Project details generated successfully")

	// Step 2: Generate File Tree
	t.Log("Step 2: Generating File Tree")
	fileTree, err := cache.Get("file_tree.txt")
	if err != nil {
		fileTree, err = llmClient.GenerateFileTree(projectDetails)
		if err != nil {
			t.Fatalf("FileTree error: %v", err)
		}
		if err := cache.Set("file_tree.txt", fileTree); err != nil {
			t.Logf("Failed to cache file tree: %v", err)
		}
	}
	t.Log("File tree generated successfully")

	// Step 3: Determine File Order
	t.Log("Step 3: Determining File Order")
	fileOrderStr, err := cache.Get("file_order.json")
	var fileOrder []string
	if err != nil {
		fileOrder, err = llmClient.DetermineFileOrder(fileTree)
		if err != nil {
			t.Fatalf("FileOrder error: %v", err)
		}
		fileOrderStr = fmt.Sprintf("%v", fileOrder)
		if err := cache.Set("file_order.json", fileOrderStr); err != nil {
			t.Logf("Failed to cache file order: %v", err)
		}
	} else {
		// Convert string back to slice
		fmt.Sscanf(fileOrderStr, "%v", &fileOrder)
	}
	t.Logf("File order determined: %v", fileOrder)

	// Step 4: Generate File Operations
	t.Log("Step 4: Generating File Operations")
	fileOperationsStr, err := cache.Get("file_operations.json")
	var fileOperations []fs.FileOperation
	if err != nil {
		fileOperations, err = llmClient.GenerateFileOperations(projectDetails, fileTree)
		if err != nil {
			t.Fatalf("FileOperations error: %v", err)
		}
		fileOperationsStr = fmt.Sprintf("%+v", fileOperations)
		if err := cache.Set("file_operations.json", fileOperationsStr); err != nil {
			t.Logf("Failed to cache file operations: %v", err)
		}
	} else {
		// Convert string back to slice of FileOperation
		// This is a simplification; you might need a more robust parsing method
		fmt.Sscanf(fileOperationsStr, "%+v", &fileOperations)
	}
	t.Logf("Generated %d file operations", len(fileOperations))

	// Step 5: Generate File Content
	t.Log("Step 5: Generating File Content for all files")
	if len(fileOrder) == 0 {
		t.Fatalf("FileOrder is empty, cannot generate file content")
	}

	fileContentMap := make(map[string]string)
	for _, fileName := range fileOrder {
		t.Logf("Generating content for file: %s", fileName)

		cacheFileName := fmt.Sprintf("file_content_%s.txt", strings.ReplaceAll(fileName, "/", "_"))
		fileContent, err := cache.Get(cacheFileName)

		if err != nil {
			fileContent, err = llmClient.GenerateFileContent(fileName, projectDetails, fileTree, fileContentMap)
			if err != nil {
				t.Fatalf("FileContent error for %s: %v", fileName, err)
			}
			if err := cache.Set(cacheFileName, fileContent); err != nil {
				t.Logf("Failed to cache file content for %s: %v", fileName, err)
			}
		}

		fileContentMap[fileName] = fileContent
		t.Logf("Generated content for file: %s", fileName)
	}

	t.Log("All steps completed successfully")
}
