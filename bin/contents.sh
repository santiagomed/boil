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

# Check if directories are provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <directory1> [directory2] ..."
    exit 1
fi

# Process all files in the specified directories
for dir in "$@"; do
    if [ ! -d "$dir" ]; then
        echo "Warning: $dir is not a directory. Skipping."
        continue
    fi
    
    find "$dir" -type f | while read -r file; do
        process_file "$file"
    done
done

echo "File contents have been copied to $OUT_FILE"