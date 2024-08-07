package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	blogger "github.com/santiagomed/boil/logger"
	"github.com/santiagomed/boil/pkg/config"
	"github.com/santiagomed/boil/pkg/core"
	"github.com/santiagomed/boil/pkg/logger"
	"github.com/santiagomed/boil/pkg/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/spf13/cobra"
)

type state int

const (
	Input state = iota
	Initializing
	Processing
	Questions
	Finished
)

type CliStepPublisher struct {
	stepChan  chan core.StepType
	errorChan chan error
	logger    logger.Logger
}

func NewCliStepPublisher(logger logger.Logger) *CliStepPublisher {
	return &CliStepPublisher{
		stepChan:  make(chan core.StepType, 100), // Buffer size of 100
		errorChan: make(chan error, 10),          // Buffer size of 10
		logger:    logger,
	}
}

func (p *CliStepPublisher) PublishStep(step core.StepType) {
	select {
	case p.stepChan <- step:
		p.logger.Debug(fmt.Sprintf("Successfully published step: %v", step))
	default:
		p.logger.Warn(fmt.Sprintf("Failed to publish step: %v. Channel full.", step))
	}
}

func (p *CliStepPublisher) Error(step core.StepType, err error) {
	select {
	case p.errorChan <- err:
		p.logger.Debug(fmt.Sprintf("Successfully published error for step: %v", step))
	default:
		p.logger.Warn(fmt.Sprintf("Failed to publish error for step: %v. Channel full.", step))
	}
}

type model struct {
	textInput       textinput.Model
	spinner         spinner.Model
	projectDesc     string
	state           state
	config          *config.Config
	currentQuestion int
	completedSteps  []core.StepType
	engine          *core.Engine
	engineCtx       context.Context
	engineCancel    context.CancelFunc
	answers         []string
	publisher       *CliStepPublisher
	logger          logger.Logger
}

type flags struct {
	name   string
	config string
}

func initialModel(prompt string, f flags) (model, error) {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	blogger.InitLogger()
	logger := blogger.GetLogger()
	logger.Debug("Initializing Boil CLI")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))

	var cfg *config.Config
	if f.config != "" {
		var err error
		cfg, err = config.LoadConfig(f.config)
		if err != nil {
			return model{}, err
		}
	} else {
		cfg = config.DefaultConfig()
	}
	if f.name != "" {
		cfg.ProjectName = f.name
	}

	projectDesc := utils.SanitizeInput(prompt)

	publisher := NewCliStepPublisher(logger)
	engine, err := core.NewProjectEngine(cfg, publisher, logger, 1)
	if err != nil {
		return model{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := model{
		textInput:       ti,
		spinner:         s,
		state:           Input,
		logger:          logger,
		projectDesc:     projectDesc,
		config:          cfg,
		engine:          engine,
		engineCtx:       ctx,
		engineCancel:    cancel,
		publisher:       publisher,
		currentQuestion: 0,
	}
	engine.Start(ctx)
	return m, nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case Input:
		return m.handleInputState(msg)
	case Questions:
		return m.handleQuestionsState(msg)
	default:
		return m.handleQuit(msg)
	}
}

func (m *model) handleInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.handleKeyEnter()
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) handleQuestionsState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.handleQuestionAnswer(m.textInput.Value())
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) handleQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
		m.logger.Debug("User exited the application")
		style := lipgloss.NewStyle().Faint(true)
		message := "Interrupted. Exiting application..."
		message = style.Render(message)
		return m, tea.Sequence(tea.Printf("%s", message), tea.Quit)
	}
	return m, nil
}

func (m *model) handleKeyEnter() (tea.Model, tea.Cmd) {
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
	return m, tea.Printf("%s", message)
}

func (m *model) handleQuestionAnswer(answer string) (tea.Model, tea.Cmd) {
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
		m.state = Initializing
		return m, func() tea.Msg { return nil }
	}

	return m, tea.Batch(textinput.Blink, func() tea.Msg { return nil })
}

func (m *model) listenForNextStep() tea.Msg {
	select {
	case step := <-m.publisher.stepChan:
		return step
	case err := <-m.publisher.errorChan:
		m.logger.Error(fmt.Sprintf("Error received during project generation: %v", err))
		return err
	}
}

func (m *model) startProjectGeneration() tea.Cmd {
	resultChan := m.engine.AddRequest(m.projectDesc)
	listenForError := func() tea.Msg {
		select {
		case err := <-resultChan:
			if err != nil {
				return err
			}
			return core.FinalizeProject
		case <-time.After(3 * time.Minute):
			m.logger.Error("Project generation timed out")
			return errors.New("project generation timed out")
		}
	}
	return tea.Batch(m.listenForNextStep, listenForError)
}

func (m *model) handleStep(step core.StepType) (tea.Model, tea.Cmd) {
	m.logger.Debug(fmt.Sprintf("Received step: %v", step))
	m.completedSteps = append(m.completedSteps, step)
	if step == core.FinalizeProject {
		m.state = Finished
		return m.finalizeProject()
	}
	return m, tea.Batch(m.spinner.Tick, m.listenForNextStep)
}

func (m *model) finalizeProject() (tea.Model, tea.Cmd) {
	m.logger.Debug("Finalizing project")
	projectName := m.config.ProjectName
	zipFileName := fmt.Sprintf("%s.zip", projectName)
	m.logger.Debug(fmt.Sprintf("Unzipping file: %s", zipFileName))

	err := utils.Unzip(zipFileName, projectName)
	if err != nil {
		m.logger.Error(fmt.Sprintf("Failed to unzip project file: %v", err))
		return m, tea.Quit
	}

	m.logger.Debug("Project unzipped successfully")

	err = os.Remove(zipFileName)
	if err != nil {
		m.logger.Error(fmt.Sprintf("Failed to delete zip file: %v", err))
		return m, tea.Quit
	}

	m.logger.Debug(fmt.Sprintf("Deleted zip file: %s", zipFileName))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	outProjectName := nameStyle.Render(projectName)
	finalMsg := fmt.Sprintf("Project generated in directory: %s", outProjectName)
	return m, tea.Printf("%s", finalMsg)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check for Finished or Initializing states
	switch m.state {
	case Finished:
		return m, tea.Quit
	case Initializing:
		m.state = Processing
		return m, tea.Batch(m.spinner.Tick, m.startProjectGeneration())
	}

	// Read the message and update the model
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m, cmd := m.handleKeyPress(msg)
		if cmd != nil {
			return m, cmd
		}
	case core.StepType:
		return m.handleStep(msg)
	case error:
		return m, tea.Sequence(tea.Printf("Error: %s", msg), tea.Quit)
	default:
		if m.state == Processing {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update the text input
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
	case Initializing:
		return fmt.Sprintf("%s Initializing", m.spinner.View())
	case Processing:
		steps := []struct {
			present string
			past    string
		}{
			{"Generating project details.", "Generated project details."},
			{"Generating file tree.", "Generated file tree."},
			{"Generating file operations.", "Generated file operations."},
			{"Executing file operations.", "Executed file operations."},
			{"Determining file order.", "Determined file order."},
			{"Generating file contents.", "Generated file contents."},
			{"Creating optional components.", "Created optional components."},
			{"Done.", "Done."},
		}

		enumerator := func(l list.Items, i int) string {
			var e string
			if i < len(m.completedSteps) {
				checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
				check := checkStyle.Render("✓")
				e = check
			} else if i == len(m.completedSteps) {
				e = m.spinner.View()
			}
			return e
		}

		l := list.New().Enumerator(enumerator)
		for i, step := range steps {
			if i < len(m.completedSteps) {
				l.Item(step.past)
			} else if i == len(m.completedSteps) {
				l.Item(step.present)
			}
		}
		return fmt.Sprint(l)
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
	case Finished:
		return "Project generated successfully!"
	default:
		m.logger.Error("An error occurred")
		return "An error occurred."
	}
}

func (m *model) updateConfig() {
	m.config.GitRepo = m.answers[0] == "y"
	m.config.GitIgnore = m.answers[1] == "y"
	m.config.Readme = m.answers[2] == "y"
	m.config.Dockerfile = m.answers[3] == "y"
}

func (m *model) Shutdown() {
	m.engineCancel()                   // Cancel the engine context
	m.engine.Shutdown(5 * time.Second) // Give 5 seconds for graceful shutdown
}

var rootCmd = &cobra.Command{
	Use:   "boil",
	Short: "Boil is a CLI tool for generating project boilerplate files",
	Long:  `Boil is a powerful CLI tool that uses AI to generate custom project boilerplate files based on your description.`,
	Run: func(cmd *cobra.Command, args []string) {
		prompt := strings.Join(args, " ")
		flags, err := parseFlags(cmd)
		if err != nil {
			fmt.Printf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		model, err := initialModel(prompt, flags)
		if err != nil {
			fmt.Printf("Error initializing model: %v\n", err)
			os.Exit(1)
		}

		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}

		model.Shutdown()
	},
}

func init() {
	rootCmd.Flags().StringP("name", "n", "", "The name of the project to generate. Also used as the project directory name")
	rootCmd.Flags().StringP("config", "c", "", "Path to custom configuration file")
}

func parseFlags(cmd *cobra.Command) (flags, error) {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return flags{}, err
	}

	config, err := cmd.Flags().GetString("config")
	if err != nil {
		return flags{}, err
	}

	return flags{
		name:   name,
		config: config,
	}, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
