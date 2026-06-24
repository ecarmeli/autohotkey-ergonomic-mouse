@echo off
echo Deploying Ergonomic Mouse for %USERNAME%...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0registerErgonomicMouseSchdTask-User.ps1"
pause