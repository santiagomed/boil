package cli

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "boil",
	Short: "Boil is a CLI tool for generating project boilerplate files",
	Long:  `Boil is a powerful CLI tool that uses AI to generate custom project boilerplate files based on your description.`,
}

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate project boilerplate files",
	Run: func(cmd *cobra.Command, args []string) {
		flags, err := parseGenFlags(cmd)
		if err != nil {
			fmt.Printf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		model, err := newGenerateModel(flags)
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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get information about existing projects",
	Run: func(cmd *cobra.Command, args []string) {
		flags, err := parseGetFlags(cmd)
		if err != nil {
			fmt.Printf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		var p *tea.Program
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBA08"))
		url := "http://localhost:8080/project/download"
		resp, err := downloadFile(url, flags.token)
		if err != nil {
			fmt.Printf("Error downloading file: %v\n", errorStyle.Render(err.Error()))
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.ContentLength <= 0 {
			fmt.Println("can't parse content length, aborting download")
			os.Exit(1)
		}

		filename := filepath.Base(resp.Request.URL.Path)
		path := filepath.Join(os.TempDir(), filename)
		file, err := os.Create(path)
		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("could not create file: %v", err)))
			os.Exit(1)
		}
		defer file.Close() // nolint:errcheck

		pw := &progressWriter{
			total:  int(resp.ContentLength),
			file:   file,
			reader: resp.Body,
			onProgress: func(ratio float64) {
				p.Send(progressMsg(ratio))
			},
		}

		m := newGetCmdModel(pw, path)

		p = tea.NewProgram(m)

		go pw.Start(p)

		if _, err := p.Run(); err != nil {
			fmt.Println("error running program:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(getCmd)

	genCmd.Flags().StringP("name", "n", "", "The name of the project to generate. Also used as the project directory name")
	genCmd.Flags().StringP("config", "c", "", "Path to custom configuration file")

	getCmd.Flags().StringP("token", "t", "", "Boil API token")
	getCmd.MarkFlagRequired("token")
}

func parseGetFlags(cmd *cobra.Command) (getFlags, error) {
	token, err := cmd.Flags().GetString("token")
	if err != nil {
		return getFlags{}, err
	}

	return getFlags{
		token: token,
	}, nil
}

func parseGenFlags(cmd *cobra.Command) (genFlags, error) {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return genFlags{}, err
	}

	config, err := cmd.Flags().GetString("config")
	if err != nil {
		return genFlags{}, err
	}

	return genFlags{
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
