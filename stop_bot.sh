#!/bin/bash
# Usage: bash stop_bot.sh
# Stops all running Go bot instances (main.go) for this project.
 
pkill -f "go run main.go"
echo "Stopped all running Go bot instances." 