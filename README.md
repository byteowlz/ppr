# PPR - Programmable Palette Renderer

PPR is a cross-platform CLI tool for creating themed wallpapers from SVG templates using base16/base24 color schemes.

## Features

- 🎨 **Theme Support**: Works with base16 and base24 color schemes
- 📐 **Custom Resolutions**: Generate wallpapers at any resolution
- 🖥️ **Auto-Detection**: Automatically detects your display resolution
- 🖼️ **Wallpaper Setting**: Cross-platform wallpaper setting (macOS, Linux, Windows)
- 📄 **SVG Templates**: Use SVG templates with color placeholders
- ⚙️ **Configurable**: TOML-based configuration system

## Installation

### From Source

```bash
git clone https://github.com/byteowlz/ppr.git
cd ppr
go build -o ppr .
```

### Binary Releases

Download the latest binary from the [releases page](https://github.com/byteowlz/ppr/releases).

## Quick Start

1. **Initialize PPR**:
   ```bash
   ppr init
   ```

2. **List available themes**:
   ```bash
   ppr list-themes
   ```

3. **Generate a wallpaper**:
   ```bash
   ppr generate --theme nord --template geometric-simple.svg
   ```

4. **Generate and set as wallpaper**:
   ```bash
   ppr generate --theme gruvbox-dark --template geometric-complex.svg --set-wallpaper
   ```

## Usage

### Commands

#### `ppr init`
Initialize PPR configuration and create necessary directories.

```bash
ppr init [--force]
```

#### `ppr list-themes`
List all available themes.

```bash
ppr list-themes [--details] [--variant dark|light]
```

#### `ppr generate`
Generate a themed wallpaper from an SVG template.

```bash
ppr generate --theme THEME --template TEMPLATE [OPTIONS]
```

**Options:**
- `--theme, -t`: Theme name to apply (required)
- `--template, -s`: Path to SVG template file (required)
- `--output, -o`: Output directory (optional)
- `--resolution, -r`: Output resolution (e.g., 1920x1080)
- `--set-wallpaper, -w`: Set generated image as wallpaper
- `--filename, -f`: Output filename (optional)

#### `ppr set-wallpaper`
Set an existing image as wallpaper.

```bash
ppr set-wallpaper IMAGE_PATH
```

### Examples

```bash
# Generate wallpaper with specific resolution
ppr generate --theme tokyo-night-storm --template geometric-simple.svg --resolution 2560x1440

# Generate and immediately set as wallpaper
ppr generate --theme catppuccin-mocha --template geometric-complex.svg --set-wallpaper

# List only dark themes with details
ppr list-themes --details --variant dark

# Generate with custom filename
ppr generate --theme nord --template geometric-simple.svg --filename my-wallpaper.png
```

## Configuration

PPR uses a TOML configuration file located at `~/.config/ppr/config.toml`:

```toml
themes_path = "/path/to/themes"
templates_path = "/path/to/templates"
output_path = "/path/to/output"
default_theme = "nord"
default_width = 1920
default_height = 1080
auto_set_wallpaper = false
```

## Creating SVG Templates

SVG templates use placeholder colors that get replaced with theme colors:

```svg
<rect fill="{{base00}}" />  <!-- Background -->
<circle fill="{{base08}}" /> <!-- Red accent -->
<path stroke="{{base0D}}" /> <!-- Blue accent -->
```

### Base16 Color Placeholders

- `{{base00}}` - Default Background
- `{{base01}}` - Lighter Background
- `{{base02}}` - Selection Background
- `{{base03}}` - Comments, Invisibles
- `{{base04}}` - Dark Foreground
- `{{base05}}` - Default Foreground
- `{{base06}}` - Light Foreground
- `{{base07}}` - Light Background
- `{{base08}}` - Red
- `{{base09}}` - Orange
- `{{base0A}}` - Yellow
- `{{base0B}}` - Green
- `{{base0C}}` - Cyan
- `{{base0D}}` - Blue
- `{{base0E}}` - Purple
- `{{base0F}}` - Brown

### Base24 Extensions

Base24 themes include additional colors `{{base10}}` through `{{base17}}`.

## Themes

PPR supports themes from the [base16](https://github.com/chriskempson/base16) and [base24](https://github.com/Base24/base24) projects. Place theme YAML files in:

```
~/.config/ppr/themes/
├── base16/
│   ├── nord.yaml
│   ├── gruvbox-dark.yaml
│   └── ...
└── base24/
    ├── catppuccin-mocha.yaml
    └── ...
```

## Platform Support

### Wallpaper Setting

- **macOS**: Uses AppleScript
- **Linux**: Supports GNOME, KDE, XFCE, i3/sway, and generic setters
- **Windows**: Uses PowerShell and Windows API

### Resolution Detection

- **macOS**: `system_profiler`
- **Linux**: `xrandr` with `xdpyinfo` fallback
- **Windows**: `wmic`

## Development

### Project Structure

```
ppr/
├── cmd/                 # CLI commands
├── pkg/
│   ├── config/         # Configuration management
│   ├── theme/          # Theme parsing and management
│   ├── svg/            # SVG template processing
│   ├── image/          # PNG generation
│   ├── resolution/     # Display resolution detection
│   └── wallpaper/      # Cross-platform wallpaper setting
├── templates/          # Example SVG templates
└── themes/            # Base16/Base24 theme files
```

### Building

```bash
go build -o ppr .
```

### Testing

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Base16](https://github.com/chriskempson/base16) project for the color scheme specification
- [Base24](https://github.com/Base24/base24) project for extended color schemes
- [oksvg](https://github.com/srwiley/oksvg) for SVG rendering
- [Cobra](https://github.com/spf13/cobra) for CLI framework