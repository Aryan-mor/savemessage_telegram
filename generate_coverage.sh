#!/bin/bash
set -e

echo "Running tests with coverage..."
go test ./... -coverprofile=coverage.out

echo "Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Done! Open coverage.html in your browser to view the report." 