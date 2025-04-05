#!/bin/bash

echo "Building sortd..."
go build -o sortd cmd/sortd/main.go

if [ $? -eq 0 ]; then
    echo "Build successful! Execute with ./sortd"
else
    echo "Build failed"
fi