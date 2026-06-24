@echo off
echo ===================================================
echo  Ergonomic Mouse Uninstaller (System-Mode)
echo ===================================================
echo.

:: 1. UAC Elevation Check
set "SCRIPT_PATH=%~f0"
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrative privileges...
    powershell -Command "Start-Process -FilePath $env:SCRIPT_PATH -Verb RunAs"
    exit /b
)
cd /d "%~dp0"

echo 1. Stopping active AutoHotkey processes...
taskkill /F /IM AutoHotkey64.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo 2. Unregistering scheduled task...
schtasks /Delete /TN "ErgonomicMouseMapping" /F >nul 2>&1

echo 3. Removing application scripts and logs...
:: Target specific files to avoid deleting unrelated scripts in the public directory
del /Q /F "%PUBLIC%\Documents\Scripts\ErgonomicMouse.ahk" >nul 2>&1
del /Q /F "%PUBLIC%\Documents\Scripts\LaunchAndUpdate.ps1" >nul 2>&1
del /Q /F "%PUBLIC%\Documents\Scripts\registerErgonomicMouseSchdTask.ps1" >nul 2>&1
del /Q /F "%PUBLIC%\Documents\Scripts\update.log" >nul 2>&1
del /Q /F "%PUBLIC%\Documents\Scripts\ErgonomicMouse.ahk.tmp" >nul 2>&1

echo.
echo Removal Complete! The system deployment has been thoroughly cleaned.
echo (Note: AutoHotkey64.exe was left installed in Program Files to protect your other active macros)
echo.
pause