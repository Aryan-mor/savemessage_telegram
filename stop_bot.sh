#!/bin/bash
# Usage: bash stop_bot.sh
# Stops all running Go bot instances (modular main.go) for this project.
 
pkill -f "go run cmd/modular/main.go"
echo "Stopped all running Go bot instances." 