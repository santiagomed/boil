package cli

import (
	"fmt"
	"os"
	"strings"

	"boil/internal/config"
	"boil/internal/generator"
	"boil/internal/llm"
	"boil/internal/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type model struct {
	p 		        *tea.Program
	textInput       textinput.Model
	spinner 	    spinner.Model
	prompt 		    string
	projectDesc     string
	outputDir       string
	state           string
	err             error
	confirmation    string
	config          *config.Config
	currentQuestion int
}

func initialModel(prompt string) model {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))


	return model{
		textInput: ti,
		spinner: s,
		prompt: prompt,
		state: "input",
		err: nil,
		confirmation: "",
		config: nil,
		currentQuestion: 0,
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
			"Project generated in temporary directory: <ADD>\n"+ // Add temporary directory path
				"Review the project and enter 'y' to finalize, or 'n' to abort: ",
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

	// Load configuration
	m.config, err = config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	m.state = "questions"
	m.currentQuestion = 0
	return nil
}

func (m *model) handleQuestions(answer string) tea.Msg {
	switch m.currentQuestion {
	case 0:
		m.config.GitRepo = strings.ToLower(answer) == "y"
	case 1:
		m.config.GitIgnore = strings.ToLower(answer) == "y"
	case 2:
		m.config.Readme = strings.ToLower(answer) == "y"
	case 3:
		m.config.License = strings.ToLower(answer) == "y"
	case 4:
		m.config.Dockerfile = strings.ToLower(answer) == "y"
	}

	m.currentQuestion++
	if m.currentQuestion >= 5 {
		return m.startProjectGeneration()
	}
	return nil
}

func (m *model) startProjectGeneration() tea.Msg {
	// Create LLM client
	llmClient := llm.NewClient(m.config)

	// Create ProjectEngine
	engine := generator.NewProjectEngine(m.config, llmClient)

	// Generate project
	err := engine.Generate(m.projectDesc)
	if err != nil {
		return fmt.Errorf("error generating project: %w", err)
	}

	// Set the temporary directory path
	m.tmpDir = m.config.TempDir

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


var rootCmd = &cobra.Command{
	Use:   "boil",
	Short: "Boil is a CLI tool for generating project boilerplate files",
	Long:  `Boil is a powerful CLI tool that uses AI to generate custom project boilerplate files based on your description.`,
	Run: func(cmd *cobra.Command, args []string) {
		prompt := strings.Join(args, " ")
		p := tea.NewProgram(initialModel(prompt))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringP("output", "o", ".", "Output directory for the generated project")
	rootCmd.Flags().StringP("config", "c", "", "Path to custom configuration file")
}