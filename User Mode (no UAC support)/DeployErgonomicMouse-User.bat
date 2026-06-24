@echo off
echo Deploying Ergonomic Mouse for %USERNAME%...

:: 1. Store the script path in an environment variable
set "DEPLOY_SCRIPT=%~dp0registerErgonomicMouseSchdTask-User.ps1"

:: 2. Use -Command and the PowerShell Call Operator (&) to unpack the envelope
powershell -NoProfile -ExecutionPolicy Bypass -Command "& $env:DEPLOY_SCRIPT"

powershell Start-Sleep -Seconds 5