package cli

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type progressMsg float64

type progressErrMsg struct{ err error }

type downloadCompleteMsg struct{}

type getFlags struct {
	token string
}

const (
	downloading = iota
	prompting
)

type getCmdModel struct {
	pw        *progressWriter
	progress  progress.Model
	path      string
	textinput textinput.Model
	state     int
	err       error
}

func newGetCmdModel(pw *progressWriter, path string) getCmdModel {
	textinput := textinput.New()
	textinput.Placeholder = "my-boil-project"
	textinput.Focus()
	textinput.CharLimit = 156
	textinput.Width = 20

	return getCmdModel{
		pw:        pw,
		progress:  progress.New(progress.WithGradient("#FFBA08", "#F48C06")),
		textinput: textinput,
		path:      path,
		state:     downloading,
	}
}

func (m getCmdModel) Init() tea.Cmd {
	return nil
}

func (m getCmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			name := m.textinput.Value()
			if name == "" {
				return m.handleSaveProject("my-boil-project")
			}
			return m.handleSaveProject(name)
		} else if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		var cmds []tea.Cmd

		if msg >= 1.0 {
			cmds = append(cmds, tea.Sequence(finalPause(), func() tea.Msg {
				return downloadCompleteMsg{}
			}))
		}

		cmds = append(cmds, m.progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	case downloadCompleteMsg:
		m.state = prompting
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m getCmdModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if m.state == prompting {
		return fmt.Sprintf("\nEnter project name: %s", m.textinput.View())
	}
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit")
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func (m getCmdModel) handleSaveProject(name string) (tea.Model, tea.Cmd) {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBA08"))
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Error getting current working directory: %v", err)))
		m.err = err
		return m, tea.Quit
	}
	destDir := filepath.Join(cwd, name)
	if err := unzip(m.path, destDir); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Error unzipping file: %v", err)))
		m.err = err
		return m, tea.Quit
	}
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	successProject := successStyle.Render(name)
	check := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	fmt.Printf("%s Project saved to directory %s\n", check, successProject)
	return m, tea.Quit
}

func downloadFile(url, token string) (*http.Response, error) {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/zip")

	// Create HTTP client with a timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Adjust timeout as needed
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("token is invalid or has expired")
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/zip" && contentType != "application/octet-stream" {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	return resp, nil
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

const (
	padding  = 2
	maxWidth = 80
)

type progressWriter struct {
	total      int
	downloaded int
	file       *os.File
	reader     io.Reader
	onProgress func(float64)
}

func (pw *progressWriter) Start(p *tea.Program) {
	// TeeReader calls pw.Write() each time a new response is received
	_, err := io.Copy(pw.file, io.TeeReader(pw.reader, pw))
	if err != nil {
		p.Send(progressErrMsg{err})
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return len(p), nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
