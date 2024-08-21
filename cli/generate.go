package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/santiagomed/boil/core"
	"github.com/santiagomed/boil/fs"
	"github.com/santiagomed/boil/logger"
	"github.com/santiagomed/boil/utils"
	"github.com/spf13/afero"
)

type state int

const (
	Input state = iota
	Initializing
	Processing
	Questions
	Finished
)

type genFlags struct {
	name   string
	config string
}

type generateCmdModel struct {
	textInput       textinput.Model
	spinner         spinner.Model
	state           state
	request         *core.Request
	currentQuestion int
	completedSteps  []core.StepType
	engine          *Engine
	engineCtx       context.Context
	engineCancel    context.CancelFunc
	answers         []string
	publisher       *CliStepPublisher
	logger          logger.Logger
	fs              *fs.FileSystem
}

func newGenerateModel(f genFlags) (generateCmdModel, error) {
	ti := textinput.New()
	ti.Placeholder = "Describe your project..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	InitLogger()
	logger := GetLogger()
	logger.Debug("Initializing Boil CLI")
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))

	req := core.DefaultRequest()
	if f.name != "" {
		req.ProjectName = f.name
	}

	fs := fs.NewMemoryFileSystem()
	publisher := NewCliStepPublisher(logger)
	engine, err := NewProjectEngine(publisher, logger, 1, fs, "http://localhost:8000")
	if err != nil {
		return generateCmdModel{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := generateCmdModel{
		textInput:       ti,
		spinner:         s,
		state:           Input,
		logger:          logger,
		request:         req,
		fs:              fs,
		engine:          engine,
		engineCtx:       ctx,
		engineCancel:    cancel,
		publisher:       publisher,
		currentQuestion: 0,
	}
	engine.Start(ctx)
	return m, nil
}

func (m generateCmdModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m generateCmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Check for Finished or Initializing states
	switch m.state {
	case Finished:
		return m, tea.Quit
	case Initializing:
		m.state = Processing
		return m, tea.Batch(m.spinner.Tick, m.handleProjectGeneration())
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

func (m generateCmdModel) View() string {
	switch m.state {
	case Input:
		return fmt.Sprintf(
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

func (m *generateCmdModel) Shutdown() {
	m.engineCancel()                   // Cancel the engine context
	m.engine.Shutdown(5 * time.Second) // Give 5 seconds for graceful shutdown
}

// handleKeyPress handles key presses for the application.
func (m *generateCmdModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case Input:
		return m.handleInputState(msg)
	case Questions:
		return m.handleQuestionsState(msg)
	default:
		return m.handleQuit(msg)
	}
}

// handleInputState handles the input state of the application on key press.
func (m *generateCmdModel) handleInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.handleKeyEnter()
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

// handleQuestionsState handles the questions state of the application on key press.
func (m *generateCmdModel) handleQuestionsState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.handleQuestionAnswer(m.textInput.Value())
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

// handleKeyEnter handles the enter key press for the application.
func (m *generateCmdModel) handleKeyEnter() (tea.Model, tea.Cmd) {
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
	m.request.ProjectDescription = v
	m.state = Questions
	placeholderStyle := lipgloss.NewStyle().Faint(true).Width(80)
	message := placeholderStyle.Render(fmt.Sprintf("> %s", v))
	return m, tea.Printf("%s", message)
}

// handleQuestionAnswer handles the question answer for the application.
func (m *generateCmdModel) handleQuestionAnswer(answer string) (tea.Model, tea.Cmd) {
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
		m.request.GitRepo = answer == "y"
	case 1:
		m.request.GitIgnore = answer == "y"
	case 2:
		m.request.Readme = answer == "y"
	case 3:
		m.request.Dockerfile = answer == "y"
	}

	m.currentQuestion++
	m.textInput.SetValue("")
	if m.currentQuestion >= 4 {
		m.request.GitRepo = m.answers[0] == "y"
		m.request.GitIgnore = m.answers[1] == "y"
		m.request.Readme = m.answers[2] == "y"
		m.request.Dockerfile = m.answers[3] == "y"
		m.state = Initializing
		return m, func() tea.Msg { return nil }
	}

	return m, tea.Batch(textinput.Blink, func() tea.Msg { return nil })
}

func (m *generateCmdModel) listenForNextStep() tea.Msg {
	select {
	case step := <-m.publisher.stepChan:
		return step
	case err := <-m.publisher.errorChan:
		m.logger.Error(fmt.Sprintf("Error received during project generation: %v", err))
		return err
	}
}

func (m *generateCmdModel) handleProjectGeneration() tea.Cmd {
	resultChan := m.engine.AddRequest(m.request)
	listenForError := func() tea.Msg {
		select {
		case err := <-resultChan:
			if err != nil {
				return err
			}
			return nil
		case <-time.After(3 * time.Minute):
			m.logger.Error("Project generation timed out")
			return errors.New("project generation timed out")
		}
	}
	return tea.Batch(m.listenForNextStep, listenForError)
}

func (m *generateCmdModel) handleStep(step core.StepType) (tea.Model, tea.Cmd) {
	m.logger.Debug(fmt.Sprintf("Received step: %v", step))
	m.completedSteps = append(m.completedSteps, step)
	if step == core.Done {
		return m.handleProjectFinalization()
	}
	return m, tea.Batch(m.spinner.Tick, m.listenForNextStep)
}

func (m *generateCmdModel) handleProjectFinalization() (tea.Model, tea.Cmd) {
	m.logger.Info("Finalizing project.")
	m.state = Finished
	projectName := utils.FormatProjectName(m.request.ProjectName)

	err := m.engine.fs.CopyDir(afero.NewOsFs(), ".", projectName)
	if err != nil {
		m.logger.Error(fmt.Sprintf("Failed to copy project to disk: %v", err))
		return m, tea.Quit
	}

	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	outProjectName := nameStyle.Render(projectName)
	finalMsg := fmt.Sprintf("Project generated in directory: %s", outProjectName)

	return m, tea.Printf("%s", finalMsg)
}

// handleQuit handles the quit state of the application on key press.
func (m *generateCmdModel) handleQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
		m.logger.Debug("User exited the application")
		style := lipgloss.NewStyle().Faint(true)
		message := "Interrupted. Exiting application..."
		message = style.Render(message)
		return m, tea.Sequence(tea.Printf("%s", message), tea.Quit)
	}
	return m, nil
}
