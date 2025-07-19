#!/bin/bash

SERVICE=telegram-bot
LOG=bot.log

case "$1" in
  reload)
    sudo systemctl daemon-reload
    ;;
  enable)
    sudo systemctl enable $SERVICE
    ;;
  start)
    sudo systemctl start $SERVICE
    ;;
  stop)
    sudo systemctl stop $SERVICE
    ;;
  restart)
    sudo systemctl restart $SERVICE
    ;;
  status)
    sudo systemctl status $SERVICE
    ;;
  monitor)
    tail -f $LOG
    ;;
  *)
    echo "Usage: $0 {reload|enable|start|stop|restart|status|monitor}"
    exit 1
    ;;
esac 