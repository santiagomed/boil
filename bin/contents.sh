#!/bin/bash

# Output file
OUT_FILE="contents.out"

# Empty the output file if it exists, or create it if it doesn't
> "$OUT_FILE"

# Function to get relative path
get_relative_path() {
    local path=$1
    echo "$path" | sed "s|^$(pwd)/||"
}

# Function to process files
process_file() {
    local file=$1
    echo "//$(get_relative_path "$file")" >> "$OUT_FILE"
    cat "$file" >> "$OUT_FILE"
    echo -e "\n" >> "$OUT_FILE"
}

# Process main.go
process_file "$(pwd)/cmd/boil/main.go"

# Process files in internal/cli
process_file "$(pwd)/internal/cli/interface.go"

# Process files in internal/config
process_file "$(pwd)/internal/config/config.go"

# Process files in internal/core
process_file "$(pwd)/internal/core/engine.go"
process_file "$(pwd)/internal/core/pipeline.go"
process_file "$(pwd)/internal/core/steps.go"

# Process files in internal/llm
process_file "$(pwd)/internal/llm/client.go"
process_file "$(pwd)/internal/llm/llm_test.go"
process_file "$(pwd)/internal/llm/prompts.go"

# Process file in internal/tempdir
process_file "$(pwd)/internal/tempdir/manager.go"

# Process files in internal/utils
process_file "$(pwd)/internal/utils/fileops.go"
process_file "$(pwd)/internal/utils/logger.go"
process_file "$(pwd)/internal/utils/sanitize.go"

echo "File contents have been copied to $OUT_FILE"