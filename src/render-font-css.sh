#!/bin/sh
set -euo pipefail

render_fontface() {
  WEIGHT="$1"
  FILE="$2"

  echo '@font-face {'
  echo 'font-family: Raleway;'
  printf 'src: local("Raleway"), url("/static/%s") format("opentype");\n' "$(basename "$FILE")"
  printf 'font-weight: %s;\n' "$WEIGHT"
  echo 'font-display: swap;'
  echo '}'
}

echo '@mixin requires-module-fonts {}'
render_fontface normal build/fonts-by-hash/raleway-regular-*.otf
render_fontface bold build/fonts-by-hash/raleway-semibold-*.otf
