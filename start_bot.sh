#!/bin/bash
# Usage: bash start_bot.sh
# Starts the Go Telegram bot and logs output to bot.log
 
go run main.go > bot.log 2>&1 