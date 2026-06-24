@echo off

:: Check for admin rights
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrative privileges...
    powershell -Command "Start-Process cmd -ArgumentList '/c \"%~f0\"' -Verb RunAs"
    exit /b
)

:: We are now elevated
echo Running PowerShell script...

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0registerErgonomicMouseSchdTask.ps1"

powershell -NoProfile -ExecutionPolicy Bypass -Command "start-sleep 5"