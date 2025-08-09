# PPR Agent Guidelines

## Build/Test Commands
- **Build**: `go build -o ppr .`
- **Test**: `go test ./...`
- **Run**: `./ppr [command]`
- **Install**: `go install .`

## Code Style
- **Language**: Go 1.24.5
- **Imports**: Standard library first, then third-party, then local packages
- **Naming**: Use camelCase for variables/functions, PascalCase for exported types
- **Error handling**: Always check and wrap errors with context using `fmt.Errorf`
- **Structs**: Use struct tags for YAML/TOML serialization (`yaml:"field"`, `toml:"field"`)
- **File organization**: Group related functionality in packages under `pkg/`

## Important
- Never use emojis in README files or commit messages.
- Keep content for README files concise and to the point

## Project Structure
- `cmd/`: CLI commands using Cobra framework
- `pkg/`: Core functionality packages (config, theme, svg, image, resolution, wallpaper)
- `templates/`: SVG template files
- Main entry point: `main.go` calls `cmd.Execute()`

## Dependencies
- CLI: `github.com/spf13/cobra`
- Config: `github.com/BurntSushi/toml`, `gopkg.in/yaml.v3`
- SVG: `github.com/srwiley/oksvg`, `github.com/srwiley/rasterx`