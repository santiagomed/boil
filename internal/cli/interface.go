package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	Listening
	Questions
	Error
)

type finished struct {
	err error
}

type CliStepPublisher struct {
	stepChan  chan core.StepType
	errorChan chan error
	logger    *zerolog.Logger
}

func NewCliStepPublisher(logger *zerolog.Logger) *CliStepPublisher {
	return &CliStepPublisher{
		stepChan:  make(chan core.StepType, 100), // Buffer size of 100
		errorChan: make(chan error, 10),          // Buffer size of 10
		logger:    logger,
	}
}

func (p *CliStepPublisher) PublishStep(step core.StepType) {
	select {
	case p.stepChan <- step:
		p.logger.Debug().Msgf("Successfully published step: %v", step)
	default:
		p.logger.Warn().Msgf("Failed to publish step: %v. Channel full.", step)
	}
}

func (p *CliStepPublisher) Error(step core.StepType, err error) {
	select {
	case p.errorChan <- err:
		p.logger.Debug().Err(err).Msgf("Successfully published error for step: %v", step)
	default:
		p.logger.Warn().Err(err).Msgf("Failed to publish error for step: %v. Channel full.", step)
	}
}

type model struct {
	textInput       textinput.Model
	spinner         spinner.Model
	prompt          string
	projectDesc     string
	state           state
	config          *config.Config
	currentQuestion int
	lastStep        core.StepType
	engine          *core.Engine
	answers         []string
	publisher       *CliStepPublisher
	logger          *zerolog.Logger
}

func initialModel(name, prompt string) (model, error) {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	logger := utils.GetLogger()

	logger.Debug().Msg("Initializing Boil CLI")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	projectDesc := utils.SanitizeInput(prompt)
	config, err := config.LoadConfig("")
	if name != "" {
		config.ProjectName = name
	}

	llmCfg := llm.LlmConfig{
		OpenAIAPIKey: config.OpenAIAPIKey,
		ModelName:    config.ModelName,
		ProjectName:  config.ProjectName,
	}
	llmClient := llm.NewClient(&llmCfg)

	publisher := NewCliStepPublisher(logger)

	engine := core.NewProjectEngine(config, llmClient, publisher, logger)

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
		publisher:       publisher,
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
		m.logger.Debug().Msg("Generating project...")
		go m.engine.Generate(m.projectDesc)
		return nil
	}
}

func (m *model) listenForSteps() tea.Cmd {
	m.logger.Debug().Msg("Listening for project generation steps")
	return func() tea.Msg {
		select {
		case step := <-m.publisher.stepChan:
			m.logger.Debug().Msgf("Received step: %v", step)
			m.lastStep = step
			if m.lastStep == core.FinalizeProjectType {
				return finished{err: nil}
			}
			return nil
		case err := <-m.publisher.errorChan:
			close(m.publisher.stepChan)
			close(m.publisher.errorChan)
			return finished{err: err}
		case <-time.After(1000 * time.Millisecond):
			// Do nothing, just wait a bit
		}
		// Schedule the next check
		return tea.Tick(1000*time.Millisecond, func(t time.Time) tea.Msg {
			return nil // or some custom message type if you prefer
		})
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
				m.logger.Debug().Msg("User exited the application")
				return m, tea.Quit
			}
		case Questions:
			switch msg.Type {
			case tea.KeyEnter:
				m.logger.Debug().Msg("User entered a response to a question")
				return m.handleQuestionsEnter(m.textInput.Value())
			case tea.KeyCtrlC, tea.KeyEsc:
				m.logger.Debug().Msg("User exited the application")
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
		m.state = Listening
		return m, m.startProjectGeneration()
	case Listening:
		return m, m.listenForSteps()
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
		return ""
	case Listening:
		steps := []struct {
			present string
			past    string
		}{
			{"Creating temporary directory.", "Created temporary directory."},
			{"Generating project details.", "Generated project details."},
			{"Generating file tree.", "Generated file tree."},
			{"Generating file operations.", "Generated file operations."},
			{"Executing file operations.", "Executed file operations."},
			{"Determining file order.", "Determined file order."},
			{"Generating file contents.", "Generated file contents."},
			{"Creating optional components.", "Created optional components."},
			{"Done.", "Done."},
		}
		var output strings.Builder
		for i, step := range steps {
			if i <= int(m.lastStep) {
				output.WriteString(fmt.Sprintf("[✔️] %s\n", step.past))
			} else if i == int(m.lastStep)+1 {
				output.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), step.present))
			}
		}

		return output.String()
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
