#!/bin/sh

rec -t raw -r 11025 -e signed -b 16 -c 1 - | go run main.go | play -t raw -r 11025 -e signed -b 16 -c 1 -
