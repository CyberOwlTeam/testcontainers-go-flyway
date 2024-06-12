#!/bin/bash

set -e

echo "run go mod tidy/verify ..."
go mod tidy
go mod verify
echo "... done"

echo "run staticcheck/vet ..."
#staticcheck ./...
go vet ./...
echo "... done"

echo "run go tests ..."
go test ./...
echo "... done"