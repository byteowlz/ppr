#!/usr/bin/env bash
set -euo pipefail

# Directories
SCHEMES="$HOME/.local/share/tinted-theming/tinty/repos/schemes"
REPOS="$HOME/.local/share/tinted-theming/tinty/repos"

# Build commands
tinted-builder-rust build "$REPOS/base16-vim/" -s "$SCHEMES"
tinted-builder-rust build "$REPOS/tinted-shell/" -s "$SCHEMES"
tinted-builder-rust build "$REPOS/tinted-terminal/" -s "$SCHEMES"
tinted-builder-rust build "$REPOS/tinted-tmux/" -s "$SCHEMES"
tinted-builder-rust build "$REPOS/tinted-yazi/" -s "$SCHEMES"
tinted-builder-rust build "$REPOS/tmux/" -s "$SCHEMES"
