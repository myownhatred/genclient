@echo off
echo Checking for updates...

git remote update
git status -uno | findstr "Your branch is behind" > nul

if %ERRORLEVEL% EQU 0 (
    echo Updates found! Pulling changes...
    git pull
    echo Rebuilding application...
    go build -o app.exe
    echo Update complete!
) else (
    echo No updates found. Already up to date.
)

pause