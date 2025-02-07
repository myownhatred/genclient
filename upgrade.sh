#!/bin/bash

echo "Checking for updates..."

# Fetch updates from remote
git remote update > /dev/null 2>&1

# Check if local is behind remote
LOCAL=$(git rev-parse @)
REMOTE=$(git rev-parse @{u})

if [ $LOCAL != $REMOTE ]; then
    echo "Updates found! Pulling changes..."
    git pull
    
    echo "Rebuilding application..."
    go build -o app
    
    if [ $? -eq 0 ]; then
        echo "Update complete!"
    else
        echo "Build failed! Please check the errors above."
        exit 1
    fi
else
    echo "No updates found. Already up to date."
fi