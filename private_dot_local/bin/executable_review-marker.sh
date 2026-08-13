#!/bin/sh
# Emit an inline review marker (REVIEW:<line> or REVIEW:<start>-<end>) for the
# helix M-r binding; picked up by Claude Code's /resolve command.
s=$1
e=$2
if [ "$s" = "$e" ]; then
   echo "REVIEW:$s"
else
   echo "REVIEW:$s-$e"
fi
