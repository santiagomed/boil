package cli

import (
	"fmt"
	"os"
	"strings"

	"boil/internal/config"
	"boil/internal/core"
	"boil/internal/llm"
	"boil/internal/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type state int

const (
	Input state = iota
	Processing
	Questions
	Confirm
	Finalizing
)

type model struct {
	textInput       textinput.Model
	spinner         spinner.Model
	prompt          string
	projectDesc     string
	tmpDir          string
	state           state
	err             error
	confirmation    string
	config          *config.Config
	currentQuestion int
	engine          *core.Engine
	answers         []string
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
		spinner:   s,
		prompt:    prompt,
		state:     Input,
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
		case Input:
			switch msg.Type {
			case tea.KeyEnter:
				m.projectDesc = m.textInput.Value()
				m.textInput.SetValue("")
				return m.setup()
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}
		case Questions:
			switch msg.Type {
			case tea.KeyEnter:
				answer := m.textInput.Value()
				m.textInput.SetValue("")
				return m.handleQuestions(answer)
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}
		case Confirm:
			switch msg.String() {
			case "y", "Y":
				m.confirmation = "y"
				m.state = Finalizing
				return m.finalizeProject()
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
	case Input:
		return fmt.Sprintf(
			"Welcome to Boil!\n\n%s\n\n%s",
			m.textInput.View(),
			"(press enter to generate project or esc to quit)",
		)
	case Processing:
		return fmt.Sprintf("%s Generating project... Please wait.", m.spinner.View())
	case Questions:
		questions := []string{
			"Do you want to initialize a git repository?",
			"Do you want to generate a .gitignore file?",
			"Do you want to generate a README.md file?",
			"Do you want to generate a Dockerfile?",
		}
		var output strings.Builder
		for i, q := range questions {
			if i < m.currentQuestion {
				output.WriteString(fmt.Sprintf("%s (%s)\n", q, m.answers[i]))
			} else if i == m.currentQuestion {
				output.WriteString(fmt.Sprintf("%s (y/n): \n%s", q, m.textInput.View()))
			}
		}
		output.WriteString("\n(Enter 'b' to go back, or 'esc' to quit)")
		return output.String()
	case Confirm:
		return fmt.Sprintf(
			"Project generated in temporary directory: %s\n"+
				"Review the project and enter 'y' to finalize, or 'n' to abort: ",
			m.tmpDir,
		)
	case Finalizing:
		return "Finalizing project..."
	default:
		return "An error occurred."
	}
}

func (m *model) setup() (tea.Model, tea.Cmd) {
	var err error

	m.projectDesc = utils.SanitizeInput(m.projectDesc)
	fmt.Println(m.projectDesc)

	m.config, err = config.LoadConfig("")
	if err != nil {
		m.err = fmt.Errorf("error loading configuration: %w", err)
	}

	llmClient := llm.NewClient(m.config)
	m.engine = core.NewProjectEngine(m.config, llmClient)

	m.state = Questions
	m.currentQuestion = 0
	m.textInput.Placeholder = "Enter y/n"
	m.textInput.CharLimit = 1

	return m, tea.Batch(textinput.Blink, func() tea.Msg { return nil })
}

func (m *model) handleQuestions(answer string) (tea.Model, tea.Cmd) {
	answer = strings.ToLower(answer)

	if answer != "y" && answer != "n" && answer != "b" {
		return m, nil
	}

	if answer == "b" && m.currentQuestion > 0 {
		m.currentQuestion--
		m.answers = m.answers[:len(m.answers)-1]
		return m, nil
	}

	m.answers = append(m.answers, answer)

	switch m.currentQuestion {
	case 0:
		m.config.GitRepo = answer == "y"
	case 1:
		m.config.GitIgnore = answer == "y"
	case 2:
		m.config.Readme = answer == "y"
	case 3:
		m.config.Dockerfile = answer == "y"
	}

	m.currentQuestion++

	if m.currentQuestion >= 5 {
		m.updateConfig()
		m.state = Processing
		return m, m.startProjectGeneration
	}

	return m, tea.Batch(textinput.Blink, func() tea.Msg { return nil })
}

func (m *model) updateConfig() {
	m.config.GitRepo = m.answers[0] == "y"
	m.config.GitIgnore = m.answers[1] == "y"
	m.config.Readme = m.answers[2] == "y"
	m.config.Dockerfile = m.answers[3] == "y"
}

func (m *model) startProjectGeneration() tea.Msg {
	var err error
	fmt.Println("generating project...")
	m.tmpDir, err = m.engine.Generate(m.projectDesc)
	if err != nil {
		return fmt.Errorf("error generating project: %w", err)
	}

	m.state = Confirm
	return nil
}

func (m *model) finalizeProject() (tea.Model, tea.Cmd) {
	err := m.engine.CleanupTempDir()
	if err != nil {
		m.err = fmt.Errorf("error finalizing project: %w", err)
		return m, tea.Quit
	}
	return m, tea.Quit
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
