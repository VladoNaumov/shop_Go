@echo off
if "%1"=="run" (
    go run ./cmd/app
) else if "%1"=="build" (
    go build -o bin/app.exe ./cmd/app
) else if "%1"=="start" (
    bin\app.exe
) else if "%1"=="clean" (
    rmdir /s /q bin
) else (
    echo Usage: make [run|build|start|clean]
)

