# Pulse

> ⚠️ **Development Status**: This project is currently in active development. Major rewrites and improvements are planned. The current version is a work in progress.

A TUI (Text User Interface) application for managing Docker stacks.

## Dependencies

- Docker API
- Required Go packages:
  - `github.com/charmbracelet/bubbletea` - For TUI interface
  - `github.com/docker/docker/client` - For Docker API interactions

## Installation

1. Make sure you have Go installed on your system
2. Clone this repository
3. Install dependencies and build:
   ```bash
   make setup
   make build
   ```

The binary will be created in the `./build` directory.

### Available Make Commands

- `make setup` - Download and verify dependencies
- `make build` - Build for current platform
- `make build-all` - Build for Linux, macOS, and Windows
- `make clean` - Remove build artifacts
- `make run` - Run the application
- `make release` - Create a release tarball

For a complete list of available commands, run:
```bash
make help
```

## Usage

Run the application:
```bash
./build/pulse
```

### Controls
- Use arrow keys to navigate through stacks
- Press 'enter' to select a stack
- In stack menu:
  - 'r' to restart stack
  - 'k' to kill stack
  - 'l' to view logs
  - 'esc' to go back
- 'q' to quit