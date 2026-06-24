@echo off
echo ===================================================
echo  Ergonomic Mouse Uninstaller (Pure Batch)
echo ===================================================
echo.

echo 1. Stopping active AutoHotkey processes...
:: /F forcefully terminates, /IM targets the image name
taskkill /F /IM AutoHotkey64.exe >nul 2>&1

:: A brief pause to ensure the OS releases any file locks
timeout /t 2 /nobreak >nul

echo 2. Unregistering scheduled task...
:: /Delete removes the task, /TN specifies the name, /F suppresses confirmation prompts
schtasks /Delete /TN "ErgonomicMouseMapping-User" /F >nul 2>&1

echo 3. Removing application files and AutoHotkey engine...
:: Check if the directory exists, then remove it quietly (/Q) and recursively (/S)
if exist "%LOCALAPPDATA%\ErgonomicMouse" (
    rmdir /S /Q "%LOCALAPPDATA%\ErgonomicMouse"
)

echo.
echo Removal Complete! Your system is clean.
echo.
pause