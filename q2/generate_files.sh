#!/bin/bash

# Check if the correct number of arguments is provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <number_of_copies>"
    exit 1
fi

# Number of copies to generate
num_copies=$1

# Source directory
source_dir="dataset"

# Check if source directory exists
if [ ! -d "$source_dir" ]; then
    echo "Error: Directory '$source_dir' does not exist."
    exit 1
fi

# Create copies of each file in the sample folder
for file in "$source_dir"/*; do
    if [ -f "$file" ]; then
        filename=$(basename -- "$file")
        extension="${filename##*.}"
        name="${filename%.*}"
        
        for ((i=1; i<=num_copies; i++)); do
            new_file="$source_dir/${name}_copy${i}.${extension}"
            cp "$file" "$new_file"
            echo "Created: $new_file"
        done
    fi
done
