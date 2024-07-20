package llm

import (
	"fmt"
)

func getProjectDetailsPrompt(projectDesc string) string {
	return fmt.Sprintf(`Based on this project description: "%s"

Generate a detailed project structure including:

1. Main Components
   - List the primary modules or components of the project
   - Briefly describe the purpose and functionality of each component

2. Dependencies and Frameworks
   - Specify main dependencies and frameworks to be used
   - Include version numbers where applicable
   - Justify the choice of each major dependency or framework

3. Configuration Requirements
   - List necessary configuration files or environment variables
   - Provide a brief description of each configuration's purpose

4. Build System
   - Recommend a build system or task runner if applicable
   - Outline any build steps or scripts that will be needed

5. Project Architecture
   - Describe the overall architecture of the project
   - Mention any specific architectural patterns or design principles to follow

6. Additional Considerations
   - Note any scalability, performance, or security considerations
   - Suggest any best practices specific to this type of project

Format your response as a structured markdown document with clear headings and subheadings.`, projectDesc)
}

func getFileTreePrompt(projectDetails string) string {
	return fmt.Sprintf(`Based on the following project details, generate a file tree structure for the project:

%s

Please provide a file tree structure that includes:
1. Essential directories for the project structure
2. Core source code files
3. Configuration files
4. README and other documentation files

Do NOT include:
1. Package manager lock files (e.g., package-lock.json, yarn.lock)
2. Build output directories (e.g., /dist, /build)
3. Dependency directories (e.g., /node_modules)
4. IDE or editor-specific files or directories
5. Temporary or cache files
6. Any files that would be automatically generated during build or runtime

Format the file tree as a text-based tree structure, using indentation to represent directory nesting. For example:

project-root/
├── .gitignore
├── README.md
├── src/
│   ├── index.js
│   ├── config/
│   │   └── config.js
│   └── utils/
│       └── helper.js
├── tests/
│   └── index.test.js
└── .env.example

Ensure the tree structure reflects a clean, production-ready project setup without including any auto-generated or temporary files.`, projectDetails)
}

func getFileOrderPrompt(fileTree string) string {
	return fmt.Sprintf(`Given the following file tree structure:

%s

Provide an ordered list of files to be created, considering dependencies and logical progression.

Return your response as a JSON object with a single key named "files", whose value is an array of file paths, relative to the project root. Each file path should be a string. Do not include any explanations or comments. For example:

{
  "files": [
    ".env",
    ".gitignore",
    "package.json",
    "package-lock.json",
    "README.md",
    "src/config/config.js",
    "src/middleware/errorHandler.js",
    "src/routes/index.js",
    "src/server.js",
    "scripts/start.js",
    "scripts/dev.js",
    "tests/server.test.js"
  ]
}

Ensure the JSON is valid and can be directly parsed. The key MUST be named "files".`, fileTree)
}

func getFileContentPrompt(filePath, projectDetails, fileTree string, previousFiles map[string]string) string {
	previousFilesContent := ""
	for path, content := range previousFiles {
		previousFilesContent += fmt.Sprintf("\n// file: %s\nContent:\n%s\n", path, content)
	}

	if previousFilesContent == "" {
		previousFilesContent = "No previous files created."
	}

	return fmt.Sprintf(`Generate the content for the file "%s" based on the following project details, file tree, and previously created files:

Project Details:
%s

File Tree:
%s

Previously Created Files:
%s

Please provide:
1. The complete content of the file, including any necessary imports or requires
2. Implement the necessary functionality based on the project details
3. Follow best practices and conventions for the given tech stack
4. Include brief comments explaining key parts of the code
5. Ensure the code is complete and ready to use (no placeholders or TODOs)
6. Make sure the content is consistent with and properly references the previously created files

Provide only the file content, without any markdown formatting or explanations outside of in-code comments.`, filePath, projectDetails, fileTree, previousFilesContent)
}

func getFileOperationsPrompt(projectDetails, fileTree string) string {
	return fmt.Sprintf(`Generate file operations for creating a project based on the following project details and file tree:

Project Details:
%s

File Tree:
%s

Please provide:
The file operation commands to create all files in the project, including both directories and files.

Format your response as a JSON object with a single key named "operations", whose value is an array of file operations. Each operation should have the following structure:
{
  "operations": [
    {
      "operation": "OPERATION_TYPE",
      "path": "relative/path/to/file/or/directory"
    },
    ...
  ]
}

Valid operation types are: CREATE_DIR, CREATE_FILE

Ensure that:
1. All necessary parent directories are created before the file
2. The operations follow the structure and conventions specified in the project details and file tree
3. No actual file content is included in these operations

The operations should only set up the file structure. File content will be generated separately.

Ensure the JSON is valid and can be directly parsed. The key MUST be named "operations".`, projectDetails, fileTree)
}

func getDockerfilePrompt(projectDetails string) string {
	return fmt.Sprintf(`Based on the following project details, generate an appropriate Dockerfile:

%s

The Dockerfile should:

1. Use an official base image appropriate for the project's primary language/framework.
2. Set up the working directory in the container.
3. Copy the necessary project files into the container.
4. Install any required dependencies.
5. Expose any necessary ports.
6. Specify the command to run the application.

Additionally, ensure that:
1. The Dockerfile follows best practices for the chosen base image and project type.
2. It includes comments explaining each significant step.
3. It optimizes for build speed and image size where possible (e.g., using multi-stage builds if appropriate).
4. Any environment-specific configurations are handled appropriately (e.g., using ARG or ENV instructions).

Return only the content of the Dockerfile, with no additional explanations or markdown formatting.`, projectDetails)
}

func getReadmePrompt(projectDetails string) string {
	return fmt.Sprintf(`Based on the following project details, generate a concise README.md file suitable for an MVP (Minimum Viable Product):

%s

The README should include the following sections:

1. Project Title
2. Brief Description
3. Key Features (limit to 3-5 core features)
4. Quick Start Guide
  - Prerequisites (keep it minimal)
  - Installation
  - Basic Usage
5. Configuration (if absolutely necessary, keep it brief)
6. Basic Troubleshooting (optional, only if there are common issues)

Format the README in Markdown. Ensure that:
1. The content is clear, concise, and focuses on getting started quickly.
2. Code snippets or commands are properly formatted in Markdown code blocks.
3. Any placeholders for project-specific information are clearly marked (e.g., [YOUR_API_KEY]).
4. The README provides just enough information for a developer to understand the project's purpose and get it running.

Omit sections on detailed API documentation, extensive testing procedures, deployment strategies, contributing guidelines, and licensing information.

Return only the content of the README.md file, formatted in Markdown.`, projectDetails)
}

func getGitignorePrompt(projectDetails string) string {
	return fmt.Sprintf(`Based on the following project details, generate an appropriate .gitignore file:

%s

The .gitignore file should:

1. Include common patterns for the primary programming language and framework used in the project.
2. Exclude common operating system files and directories (e.g., .DS_Store for macOS, Thumbs.db for Windows).
3. Ignore dependency directories (e.g., node_modules for Node.js projects).
4. Exclude build output and compiled files.
5. Ignore common IDE and text editor specific files and directories.
6. Exclude log files and other runtime-generated files.
7. Ignore environment-specific files (e.g., .env files containing sensitive information).

Additionally:
- Group related ignore patterns together with clear comments.
- Include patterns for both directories and files where applicable.
- Use wildcard patterns judiciously to cover variations (e.g., *.log for all log files).
- If the project uses a specific build tool or package manager, include relevant ignore patterns for those.

Return only the content of the .gitignore file, with each pattern on a new line and using # for comments.`, projectDetails)
}
