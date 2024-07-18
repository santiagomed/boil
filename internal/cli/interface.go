package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/santiagomed/boil/internal/generator"
	"github.com/santiagomed/boil/internal/llm"
	"github.com/santiagomed/boil/internal/tempdir"
	"github.com/santiagomed/boil/pkg/utils"
)

type model struct {
	textInput    textinput.Model
	projectDesc  string
	outputDir    string
	tmpDir       string
	state        string
	err          error
	confirmation string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	return model{
		textInput: ti,
		outputDir: ".",
		state:     "input",
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "input":
			switch msg.Type {
			case tea.KeyEnter:
				m.projectDesc = m.textInput.Value()
				m.state = "processing"
				return m, m.generateProject
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}
		case "confirm":
			switch msg.String() {
			case "y", "Y":
				m.confirmation = "y"
				m.state = "finalizing"
				return m, m.finalizeProject
			case "n", "N", "q", "Q":
				return m, tea.Quit
			}
		}
	case error:
		m.err = msg
		return m, tea.Quit
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	switch m.state {
	case "input":
		return fmt.Sprintf(
			"Welcome to Boil!\n\n%s\n\n%s",
			m.textInput.View(),
			"(press enter to generate project or esc to quit)",
		)
	case "processing":
		return "Generating project... Please wait."
	case "confirm":
		return fmt.Sprintf(
			"Project generated in temporary directory: %s\n"+
				"Review the project and enter 'y' to finalize, or 'n' to abort: ",
			m.tmpDir,
		)
	case "finalizing":
		return "Finalizing project..."
	default:
		return "An error occurred."
	}
}

func (m *model) generateProject() tea.Msg {
	var err error

	// Sanitize input
	m.projectDesc = utils.SanitizeInput(m.projectDesc)

	// Create temporary directory
	m.tmpDir, err = tempdir.CreateTempProjectDir()
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %v", err)
	}

	// Generate project steps and details
	projectDetails, err := llm.GenerateProjectDetails(m.projectDesc)
	if err != nil {
		return fmt.Errorf("error generating project details: %v", err)
	}

	// Generate file tree
	fileTree, err := llm.GenerateFileTree(projectDetails)
	if err != nil {
		return fmt.Errorf("error generating file tree: %v", err)
	}

	// Determine file creation order
	fileOrder, err := llm.DetermineFileOrder(fileTree)
	if err != nil {
		return fmt.Errorf("error determining file creation order: %v", err)
	}

	// Generate code for each file
	for _, file := range fileOrder {
		fileContent, err := llm.GenerateFileContent(file, projectDetails, fileTree)
		if err != nil {
			return fmt.Errorf("error generating content for %s: %v", file, err)
		}

		err = generator.CreateFile(m.tmpDir, file, fileContent)
		if err != nil {
			return fmt.Errorf("error creating file %s: %v", file, err)
		}
	}

	m.state = "confirm"
	return nil
}

func (m *model) finalizeProject() tea.Msg {
	err := generator.FinalizeProject(m.tmpDir, m.outputDir)
	if err != nil {
		return fmt.Errorf("error finalizing project: %v", err)
	}
	return tea.Quit
}

func Execute() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}