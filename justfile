# Justfile for task-herald

# Build the Go project
build:
  go build -o bin/task-herald ./cmd

# Run the Go project (builds first)
run:
  just build
  ./bin/task-herald
