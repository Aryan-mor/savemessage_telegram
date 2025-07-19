#!/bin/bash
# Usage: bash restart_bot.sh
# Stops all running Go bot instances, starts the bot, and tails the log.
 
bash stop_bot.sh
bash start_bot.sh &
sleep 2
bash tail_log.sh