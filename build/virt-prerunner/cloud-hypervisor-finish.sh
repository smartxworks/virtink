#!/bin/sh

if test "$1" -eq 256 ; then
  e=$((128 + $2))
else
  e="$1"
fi

echo "$e" > /run/s6-linux-init-container-results/exitcode
/run/s6/basedir/bin/halt
