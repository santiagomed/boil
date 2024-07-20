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
	return fmt.Sprintf(`Based on the following project details:

%s

Generate a comprehensive file tree structure that:

1. Reflects all main components and modules described
2. Follows conventions for the specified tech stack and frameworks
3. Organizes code logically (e.g., separate routes, models, controllers, services)
4. Includes all necessary configuration files, build scripts, and dotfiles
5. Represents test directories, documentation, and any other project-specific needs
6. Uses appropriate naming conventions for the primary programming language(s)

Present the file tree as plain text, using indentation to show hierarchy. Use '/' for directories and no trailing slash for files. For example:

src/
 main.js
 components/
   Header.js
   Footer.js
config/
 database.js

Provide only the file tree structure, with no additional explanations or comments.`, projectDetails)
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
