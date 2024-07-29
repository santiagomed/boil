# Boil

Boil is an AI-powered CLI tool that generates custom project boilerplate code based on your project description.

## Features

- **AI-Driven Code Generation**: Describe your project, and Boil creates tailored boilerplate code.
- **Multi-Language Support**: Works with various programming languages and frameworks.
- **Customizable Output**: Generate key project components including file structure, core source files, and configuration files.
- **Optional Components**: Choose to include git initialization, .gitignore, README.md, and Dockerfile.
- **Interactive CLI**: User-friendly command-line interface for easy project customization.

## Installation

You can install Boil using Homebrew with the santiagomed/tap:

```bash
brew tap santiagomed/tap
brew install boil
```

## Usage

To use Boil, simply run:

```bash
boil
```

Boil will prompt you to enter a description of your project. After you provide the description, it will generate the boilerplate code based on your input.

### Options

- `--name, -n`: Set the project name (also used as the directory name)
- `--config, -c`: Specify a custom configuration file path

For more options:

```bash
boil --help
```

## Configuration

**Note: Configuration files are currently a work in progress and might not function as expected.**

In the future, Boil will be configurable using:

1. Command-line flags
2. Environment variables (prefixed with `BOIL_`)
3. Configuration file (`config.yaml` in the current directory or `~/.boil/config.yaml`)

Example configuration file (not yet implemented):

```yaml
project_name: my-awesome-project
openai_api_key: your-api-key-here
model_name: gpt-4-turbo-preview
git_repo: true
git_ignore: true
readme: true
dockerfile: false
```

For now, please use command-line options to customize Boil's behavior.

## Examples

Generate a simple Express.js web server:

```bash
Welcome to Boil!

>  simple web server in Express that returns 'Hello, World!' when '/' is accessed.

(press enter to generate project or esc to quit)
```

## Contributing

We welcome contributions to Boil! Please see our [Contributing Guidelines](CONTRIBUTING.md) for more details.

## License

Boil is released under the [GNU General Public License v3.0 (GPL-3.0)](https://www.gnu.org/licenses/gpl-3.0.en.html).

## Support

If you encounter any issues or have questions, please file an issue on our [GitHub repository](https://github.com/santiagomed/boil/issues).

---

Boil is currently in beta. We appreciate your feedback and patience as we continue to improve the tool.