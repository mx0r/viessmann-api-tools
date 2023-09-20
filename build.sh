#!/bin/sh

# Build linux and macOS versions

echo Building linux version...
GOOS=linux GOARCH=amd64 go build -o bin/vsgetfeats.linux vsgetfeats.go
echo Building macOS version...
GOOS=darwin GOARCH=amd64 go build -o bin/vsgetfeats.macos vsgetfeats.go
echo Done.