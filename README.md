# proompting
Get up and vibe coding quickly

## Overview
Proompting provides a framework for managing complex projects with AI-powered task execution. It includes prompt templates that help structure project planning and task management for maximum efficiency and clarity.

## Requirements
- Git
- Bash/Zsh shell
- `GIT_DIRECTORY` environment variable must be set to your git projects directory
  ```bash
  export GIT_DIRECTORY="$HOME/git"  # or wherever you keep your git projects
  ```

## Installation
1. Clone this repository:
   ```bash
   cd $GIT_DIRECTORY
   git clone <repository-url> proompting
   cd proompting
   ```

2. The main script is already executable and ready to use.

## Usage

### Setting up proompts in another project
Use the `get-the-vibes-going.sh` script to copy the proompting templates to any git project:

```bash
# Set up your environment (add to ~/.bashrc or ~/.zshrc)
export GIT_DIRECTORY="$HOME/git"

# Run from the proompting directory
./get-the-vibes-going.sh $GIT_DIRECTORY/your-project-name

# Or with a relative path
./get-the-vibes-going.sh ../another-project

# Or with an absolute path
./get-the-vibes-going.sh /path/to/any/git/project
```

The script will:
- ✅ Validate the target is a git repository
- ✅ Create a `proompts` directory in the target project
- ✅ Copy all prompt templates from this project
- ✅ Add `proompts/` to the target's `.gitignore` (creates one if needed)
- ✅ Provide clear feedback on what was done

### What's in proompts?
The `proompts` directory contains specialized markdown templates:

- **`initial-prompt.md`**: A comprehensive template for creating detailed project task breakdowns. Use this when starting a new project to generate a structured `tasks.yaml` file.

- **`get-next-task.md`**: An execution framework for tackling the next priority task. This prompt helps maintain momentum and ensures thorough, high-quality task completion.

## Example Workflow
1. Set up proompts in your project:
   ```bash
   ./get-the-vibes-going.sh $GIT_DIRECTORY/my-new-project
   ```

2. Use the templates with your AI assistant to:
   - Generate a comprehensive project plan
   - Break down complex tasks into manageable pieces
   - Execute tasks with precision and thoroughness

## Notes
- The `proompts` directory is automatically added to `.gitignore` to keep your project-specific prompts local
- All file references in templates use `$GIT_DIRECTORY` to maintain portability
- The templates are designed for use with AI assistants that support complex project management
