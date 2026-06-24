@echo off

:: Store the script path in an environment variable to bypass quote/parenthesis parsing traps
set "SCRIPT_PATH=%~f0"

:: Check for admin rights
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrative privileges...
    powershell -Command "Start-Process -FilePath $env:SCRIPT_PATH -Verb RunAs"
    exit /b
)

:: We are now elevated. 
cd /d "%~dp0"

echo Running PowerShell script...

powershell -NoProfile -ExecutionPolicy Bypass -File "registerErgonomicMouseSchdTask.ps1"

powershell Start-Sleep -Seconds 5