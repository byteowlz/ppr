![ppr_banner](ppr_logo_banner_black.png)

# PPR - Programmable Palette Renderer

PPR is a cross-platform CLI tool for creating themed wallpapers from SVG templates using base16/base24 color schemes.

## Features

- **Theme Support**: Works with base16 and base24 color schemes
- **Color Extraction**: Extract color schemes from existing SVG files to create new themes
- **Template Cycling**: Cycle through preferred templates with automatic wallpaper setting
- **Custom Resolutions**: Generate wallpapers at any resolution
- **Auto-Detection**: Automatically detects your display resolution
- **Wallpaper Setting**: Cross-platform wallpaper setting (macOS, Linux, Windows)
- **SVG Templates**: Use SVG templates with color placeholders
- **Configurable**: TOML-based configuration system

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

3. **List available templates**:

   ```bash
   ppr list-templates
   ```

4. **Generate a wallpaper**:

   ```bash
   ppr generate --theme nord --template shapes
   ```

5. **Generate and set as wallpaper**:

   ```bash
   ppr generate --theme gruvbox-dark --template shapes --set-wallpaper
   ```

6. **Cycle through templates**:

   ```bash
   ppr cycle nord  # Cycles to next template and sets as wallpaper
   ```

7. **Extract colors from SVG**:

   ```bash
   ppr extract-colors example/example.svg my-custom-theme
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

#### `ppr list-templates`

List all available SVG templates.

```bash
ppr list-templates [--details]
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

#### `ppr cycle`

Cycle through preferred templates and set as wallpaper.

```bash
ppr cycle [THEME_NAME]
```

**Options:**
- `--set-wallpaper, -w`: Set generated image as wallpaper (default: true)
- `--output, -o`: Output directory (optional)
- `--filename, -f`: Output filename (optional)
- `--resolution, -r`: Output resolution (e.g., 1920x1080)
- `--svg`: Output SVG file instead of PNG

**Note**: The cycle command always sets the wallpaper by default, making it perfect for quick theme switching.

#### `ppr extract-colors`

Extract color scheme from SVG file and create a new theme.

```bash
ppr extract-colors <svg-file> <theme-name>
```

This command analyzes an SVG file containing color swatches labeled with base00-base0F and creates a new theme file. Perfect for converting visual color palettes into usable themes.

#### `ppr set-wallpaper`

Set an existing image as wallpaper.

```bash
ppr set-wallpaper IMAGE_PATH
```

### Examples

```bash
# List available templates
ppr list-templates

# List templates with details
ppr list-templates --details

# Generate wallpaper with specific resolution
ppr generate --theme tokyo-night-storm --template shapes --resolution 2560x1440

# Generate and immediately set as wallpaper
ppr generate --theme catppuccin-mocha --template shapes --set-wallpaper

# Cycle through templates with Nord theme
ppr cycle nord

# Extract colors from a custom SVG and create new theme
ppr extract-colors my-colors.svg my-custom-theme

# List only dark themes with details
ppr list-themes --details --variant dark

# Generate with custom filename
ppr generate --theme nord --template shapes --filename my-wallpaper.png
```

## Configuration

PPR uses a TOML configuration file located at `~/.config/ppr/config.toml`:

```toml
themes_path = "/path/to/themes"
templates_path = "/path/to/templates"
output_path = "/path/to/output"
default_theme = "nord"
default_template = "shapes.svg"
default_width = 1920
default_height = 1080
auto_set_wallpaper = false
current_theme = "nord"
current_template = "shapes.svg"
last_output_path = "/path/to/last/generated/image.png"
preferred_templates = ["all"]  # or ["shapes", "horizontal_bar", "vertical_bar"]
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

## Creating Custom Color Schemes

PPR provides tools to easily create your own color schemes:

### Method 1: Extract from Visual SVG

1. **Start with the example**: Copy `example/example.svg` and modify the colors to your liking
2. **Extract the colors**: Use `ppr extract-colors` to create a theme file
3. **Use your theme**: Generate wallpapers with your custom color scheme

```bash
# Copy and edit the example
cp example/example.svg my-colors.svg
# Edit my-colors.svg with your preferred colors

# Extract the color scheme
ppr extract-colors my-colors.svg my-theme

# Use your new theme
ppr generate --theme my-theme --template shapes --set-wallpaper
```

### Method 2: Use the Color Template

Generate a color reference sheet with any existing theme:

```bash
# Generate a color reference using the base16-colors-template
ppr generate --theme nord --template base16-colors-template --filename nord-colors.png
```

This creates a visual reference showing all 16 colors with their labels, perfect for:
- Understanding color relationships in a theme
- Creating color documentation
- Sharing color palettes visually

### Supported SVG Formats for Extraction

The `extract-colors` command supports various SVG formats:

- **Simple format**: Direct `fill` attributes with `<text>` labels
- **Complex format**: CSS classes with `<tspan>` text elements
- **Mixed format**: Combination of both approaches

The command automatically detects base colors (base00-base0F) from text labels and matches them with corresponding fill colors.

### Example Files Included

PPR includes several example files to get you started:

- **`example/example.svg`**: Nord color palette in simple format - perfect for copying and modifying
- **`example/dmg_dark.svg`**: Same Nord colors in complex CSS format - demonstrates compatibility
- **`example/base16-colors-template.svg`**: Template that displays all 16 colors with labels for any theme

Try them out:

```bash
# Extract from the simple example
ppr extract-colors example/example.svg my-nord-copy

# Extract from the complex example  
ppr extract-colors example/dmg_dark.svg another-nord-copy

# Generate a color reference sheet
ppr generate --theme catppuccin-mocha --template base16-colors-template --filename mocha-colors.png
```

## Themes

PPR supports themes from the [base16](https://github.com/chriskempson/base16) and [base24](https://github.com/Base24/base24) projects. Place theme YAML files in:

```
~/.config/ppr/themes/
├── base16/
│   ├── nord.yaml
│   ├── gruvbox-dark.yaml
│   ├── my-custom-theme.yaml  # Your extracted themes
│   └── ...
└── base24/
    ├── catppuccin-mocha.yaml
    └── ...
```

### Theme File Format

Extracted themes follow the standard Base16 YAML format:

```yaml
system: "base16"
name: "my-theme"
author: "extracted"
variant: "dark"
palette:
  base00: "#2E3440"  # Background
  base01: "#3B4252"  # Lighter Background
  # ... (base02-base0F)
  base0F: "#5E81AC"  # Brown
```

## Template Cycling

The `cycle` command allows you to easily rotate through your favorite templates:

### Configuration

Set your preferred templates in `config.toml`:

```toml
# Cycle through all available templates
preferred_templates = ["all"]

# Or specify a custom list
preferred_templates = ["shapes", "horizontal_bar", "vertical_bar", "shapes_overlap"]
```

### Usage

```bash
# Cycle to next template with current theme
ppr cycle

# Cycle with specific theme
ppr cycle nord

# Cycle without setting wallpaper
ppr cycle --set-wallpaper=false
```

The cycle command:
- Automatically moves to the next template in your preferred list
- Sets the wallpaper by default (can be disabled)
- Remembers the current position for seamless cycling
- Wraps around to the first template after the last one

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
│   ├── extract-colors.go  # Color extraction from SVG
│   ├── cycle.go           # Template cycling
│   └── ...
├── pkg/
│   ├── config/         # Configuration management
│   ├── theme/          # Theme parsing and management
│   ├── svg/            # SVG template processing
│   ├── image/          # PNG generation
│   ├── resolution/     # Display resolution detection
│   └── wallpaper/      # Cross-platform wallpaper setting
├── example/            # Example SVG files for color extraction
│   ├── example.svg     # Nord color palette example
│   ├── dmg_dark.svg    # Complex SVG format example
│   └── base16-colors-template.svg  # Color reference template
├── pkg/templates/      # Built-in SVG templates
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
