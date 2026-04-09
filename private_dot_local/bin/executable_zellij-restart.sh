#!/bin/bash

LAYOUT_DIR=$(mktemp -d /tmp/zellij-layouts-XXXXXX)

sessions=$(zellij list-sessions | awk '{print $1}')

if [ -z "$sessions" ]; then
  echo "No active zellij sessions found."
  exit 1
fi

for session in $sessions; do
  echo "Dumping layout for session: $session"
  zellij -s "$session" action dump-layout > "$LAYOUT_DIR/$session.kdl" 2>/dev/null
done

echo "Killing all sessions..."
zellij kill-all-sessions -y
sleep 1

for session in $sessions; do
  echo "Relaunching session: $session"
  zellij -s "$session" --layout "$LAYOUT_DIR/$session.kdl" 2>/dev/null &
done

sleep 2
rm -rf "$LAYOUT_DIR"
echo "Done. Sessions ready — attach with: zellij attach <name>"
