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
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type state int

const (
	Input state = iota
	Processing
	Questions
	Error
)

type model struct {
	textInput       textinput.Model
	spinner         spinner.Model
	prompt          string
	projectDesc     string
	state           state
	config          *config.Config
	currentQuestion int
	engine          *core.Engine
	answers         []string
	logger          *zerolog.Logger
}

type finished struct {
	err error
}

func initialModel(name, prompt string) (model, error) {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	logger := utils.GetLogger()

	logger.Info().Msg("Initializing Boil CLI")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	projectDesc := utils.SanitizeInput(prompt)
	config, err := config.LoadConfig("")
	if name != "" {
		config.ProjectName = name
	}

	llmClient := llm.NewClient(config)
	engine := core.NewProjectEngine(config, llmClient, logger)

	if err != nil {
		return model{}, fmt.Errorf("error loading configuration: %w", err)
	}

	return model{
		textInput:       ti,
		spinner:         s,
		prompt:          prompt,
		state:           Input,
		logger:          logger,
		projectDesc:     projectDesc,
		config:          config,
		engine:          engine,
		currentQuestion: 0,
	}, nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) handleInputEnter() (tea.Model, tea.Cmd) {
	if m.state != Input {
		return m, nil
	}
	v := m.textInput.Value()

	// No input, quit.
	if v == "" {
		placeholderStyle := lipgloss.NewStyle().Faint(true)
		message := "No project description entered. Exiting..."
		message = placeholderStyle.Render(message)
		return m, tea.Sequence(tea.Printf("%s", message), tea.Quit)
	}
	// Input, run query.
	m.textInput.SetValue("")
	m.projectDesc = v
	m.state = Questions
	placeholderStyle := lipgloss.NewStyle().Faint(true).Width(80)
	message := placeholderStyle.Render(fmt.Sprintf("> %s", v))
	return m, tea.Sequence(tea.Printf("%s", message), m.spinner.Tick)
}

func (m *model) handleQuestionsEnter(answer string) (tea.Model, tea.Cmd) {
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
	m.textInput.SetValue("")

	if m.currentQuestion >= 4 {
		m.updateConfig()
		m.state = Processing
		return m, func() tea.Msg { return nil }
	}

	return m, tea.Batch(textinput.Blink, func() tea.Msg { return nil })
}

func (m *model) startProjectGeneration() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info().Msg("Generating project...")
		err := m.engine.Generate(m.projectDesc)
		return finished{err}
	}
}

func (m *model) handleOutput(err error) (tea.Model, tea.Cmd) {
	if err != nil {
		m.logger.Error().Err(err).Msg("Error generating project")
		return m, tea.Sequence(tea.Printf("Error generating project: %s", err), tea.Quit)
	}

	placeholderStyle := lipgloss.NewStyle().Faint(true)
	message := fmt.Sprintf("Project generated in directory: %s", m.config.ProjectName)
	message = placeholderStyle.Render(message)
	m.engine.CleanupTempDir()
	return m, tea.Sequence(tea.Printf("%s", message), tea.Quit)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case Input:
			switch msg.Type {
			case tea.KeyEnter:
				return m.handleInputEnter()
			case tea.KeyCtrlC, tea.KeyEsc:
				m.logger.Info().Msg("User exited the application")
				return m, tea.Quit
			}
		case Questions:
			switch msg.Type {
			case tea.KeyEnter:
				m.logger.Debug().Msg("User entered a response to a question")
				return m.handleQuestionsEnter(m.textInput.Value())
			case tea.KeyCtrlC, tea.KeyEsc:
				m.logger.Info().Msg("User exited the application")
				return m, tea.Quit
			}
		}
	case finished:
		return m.handleOutput(msg.err)
	case error:
		return m, tea.Sequence(tea.Printf("Error: %s", msg), tea.Quit)
	}

	switch m.state {
	case Processing:
		return m, m.startProjectGeneration()
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
	default:
		m.logger.Error().Msg("An error occurred")
		return "An error occurred."
	}
}

func (m *model) updateConfig() {
	m.config.GitRepo = m.answers[0] == "y"
	m.config.GitIgnore = m.answers[1] == "y"
	m.config.Readme = m.answers[2] == "y"
	m.config.Dockerfile = m.answers[3] == "y"
}

var rootCmd = &cobra.Command{
	Use:   "boil",
	Short: "Boil is a CLI tool for generating project boilerplate files",
	Long:  `Boil is a powerful CLI tool that uses AI to generate custom project boilerplate files based on your description.`,
	Run: func(cmd *cobra.Command, args []string) {
		prompt := strings.Join(args, " ")
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		model, err := initialModel(name, prompt)
		if err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringP("name", "n", "", "The name of the project to generate. Also used as the project directory name")
	rootCmd.Flags().StringP("config", "c", "", "Path to custom configuration file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
