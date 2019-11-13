#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
shift

mkdir -p "${TARGET_DIR}/"

sha256() {
  sha256sum | cut -c1-64
}

for INPUT_FILE in "$@"; do
  HASH="$(cat "$INPUT_FILE" | sha256)"

  # construct output filename containing a hash of the content,
  # e.g. "foo.min.js" -> "foo-$HASH.min.js"
  #
  # BUT: if input filename already contains hash, do not add it again
  FILENAME="$(basename "$INPUT_FILE")"
  BASENAME="${FILENAME%%.*}"
  EXT="${FILENAME#*.}"
  OUTPUT_FILE="${TARGET_DIR}/${BASENAME%%-${HASH}}-${HASH}.${EXT}"

  # create output file only if it does not exist yet
  if [ ! -f "${OUTPUT_FILE}" ]; then
    printf "\t\e[37m%s -> %s\e[0m\n" "${INPUT_FILE}" "${OUTPUT_FILE}"
    cp "${INPUT_FILE}" "${OUTPUT_FILE}"
  fi
  # if it's a JS bundle with a source map, also copy the source map
  if [ -f "${INPUT_FILE}.map" -a ! -f "${OUTPUT_FILE}.map" ]; then
    printf "\t\e[37m%s -> %s\e[0m\n" "${INPUT_FILE}.map" "${OUTPUT_FILE}.map"
    cp "${INPUT_FILE}.map" "${OUTPUT_FILE}.map"
  fi

  # keep track that we imported this file
  VAR_NAME="KEEP_$(echo "$OUTPUT_FILE" | sha256)"
  eval "$VAR_NAME=1"
  if [ -f "${INPUT_FILE}.map" ]; then
    VAR_NAME="KEEP_$(echo "$OUTPUT_FILE.map" | sha256)"
    eval "$VAR_NAME=1"
  fi
done

# remove all unwanted files from ${TARGET_DIR}
for OUTPUT_FILE in "${TARGET_DIR}"/*; do
  VAR_NAME="KEEP_$(echo "$OUTPUT_FILE" | sha256)"
  if [ "${!VAR_NAME:-0}" != "1" ]; then
    rm -- "${OUTPUT_FILE}"
  fi
done
