package llm

import "fmt"

func getProjectDetailsPrompt(projectDesc string) string {
	return fmt.Sprintf(`Create a detailed plan for a software project based on this description: "%s"

Please provide:
1. A list of main components/features of the project
2. A step-by-step guide for setting up the project
3. Key technologies and libraries to be used
4. Basic structure of the database (tables and their relationships)
5. Main API endpoints to be implemented
6. Any additional considerations or best practices to keep in mind

Format your response as a structured markdown document with clear headings and subheadings.`, projectDesc)
}

func getFileTreePrompt(projectDetails string) string {
	return fmt.Sprintf(`Based on the following project details, generate a comprehensive file tree structure:

%s

Please provide a file tree structure that:
1. Includes all necessary directories and files
2. Follows conventions for the specified tech stack
3. Organizes code logically (e.g., separate routes, models, controllers)
4. Includes configuration files, database scripts, and documentation files

Format your response as a text-based tree structure.`, projectDetails)
}

func getFileOrderPrompt(fileTree string) string {
	return fmt.Sprintf(`Given the following file tree structure, provide an ordered list of files to be created, considering dependencies and logical progression:

%s

Please provide:
1. An ordered list of files to be created, with each file on a new line
2. Brief explanations for any non-obvious ordering decisions

Format your response as a numbered list of file paths, relative to the project root.`, fileTree)
}

func getFileContentPrompt(filePath, projectDetails, fileTree string) string {
	return fmt.Sprintf(`Generate the content for the file "%s" based on the following project details and file tree:

Project Details:
%s

File Tree:
%s

Please provide:
1. The complete content of the file, including any necessary imports or requires
2. Implement the necessary functionality based on the project details
3. Follow best practices and conventions for the given tech stack
4. Include brief comments explaining key parts of the code
5. Ensure the code is complete and ready to use (no placeholders or TODOs)

Provide only the file content, without any markdown formatting or explanations outside of in-code comments.`, filePath, projectDetails, fileTree)
}

func getFileOperationsPrompt(filePath, projectDetails, fileTree string) string {
	return fmt.Sprintf(`Generate file operations for creating the file "%s" based on the following project details and file tree:

Project Details:
%s

File Tree:
%s

Please provide:
1. The file operation commands to create this file and any necessary parent directories
2. The content of the file

Format your response as a JSON array of file operations, where each operation has the following structure:
{
  "operation": "OPERATION_TYPE",
  "path": "relative/path/to/file/or/directory",
  "content": "File content goes here (for CREATE_FILE operations)"
}

Valid operation types are: CREATE_DIR, CREATE_FILE

Ensure that:
1. The file content includes appropriate imports/requires
2. The necessary functionality is implemented based on the project details
3. Best practices and conventions for the given tech stack are followed
4. Brief comments explaining key parts of the code are included
5. The code is complete and ready to use (no placeholders or TODOs)`, filePath, projectDetails, fileTree)
}