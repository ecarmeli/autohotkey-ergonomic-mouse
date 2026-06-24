@echo off
setlocal enabledelayedexpansion
title Ergonomic Mouse Unified Manager

:: =========================================================================
:: 1. Initialization & Routing
:: =========================================================================
:: Fix the working directory trap
cd /d "%~dp0"
set "SCRIPT_PATH=%~f0"

:: If relaunched by UAC for System Mode, jump directly to the elevated menu
if "%~1"=="/System" goto SYSTEM_MENU


:: =========================================================================
:: 2. Main Menu
:: =========================================================================
:MAIN_MENU
call :CHECK_STATE
set "MODE=!ACTIVE_MODE!"
cls
echo ===================================================
echo  Ergonomic Mouse Manager (Unified)
echo ===================================================
echo.

echo  Current Detection Status:
if "!SYS_INSTALLED!"=="1" ( echo   - System-Mode : [INSTALLED] ) else ( echo   - System-Mode : [Not Installed] )
if "!USER_INSTALLED!"=="1" ( echo   - User-Mode   : [INSTALLED] ) else ( echo   - User-Mode   : [Not Installed] )

echo.

if "!ACTIVE_MODE!"=="SYSTEM" (
    echo  NOTE: System-Mode is active.
    echo        To switch to User-Mode, uninstall System-Mode first.
)

if "!ACTIVE_MODE!"=="USER" (
    echo  NOTE: User-Mode is active.
    echo        To switch to System-Mode, uninstall User-Mode first.
)

if "!ACTIVE_MODE!"=="CONFLICT" (
    echo  WARNING: Both System-Mode and User-Mode appear installed.
    echo           Please uninstall one mode before proceeding.
)

echo.
echo ===================================================

:: Dynamic menu construction
set "MENU_CHOICES="
set "OPT_SYS="
set "OPT_USER="
set "OPT_EXIT="
set "IDX=1"

:: System option
if "!ACTIVE_MODE!"=="USER" goto SYS_LOCKED

echo  [!IDX!] Manage System-Mode Deployment (Requires Admin)
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_SYS=!IDX!"
set /a IDX+=1
goto SYS_DONE

:SYS_LOCKED
echo  [ ] Manage System-Mode Deployment - (Locked: Remove User-Mode First)

:SYS_DONE

:: User option
if /I "!ACTIVE_MODE!"=="SYSTEM" goto USER_LOCKED

echo  [!IDX!] Manage User-Mode Deployment
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_USER=!IDX!"
set /a IDX+=1
goto USER_DONE

:USER_LOCKED
echo  [ ] Manage User-Mode Deployment - (Locked: Remove System-Mode First)

:USER_DONE

:: Exit option (always available)
echo  [!IDX!] Exit
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_EXIT=!IDX!"

echo.
choice /C !MENU_CHOICES! /N /M "Select an option: "

:: Routing (in descending priority order!)
if defined OPT_EXIT if errorlevel !OPT_EXIT! goto END
if defined OPT_USER if errorlevel !OPT_USER! goto USER_MENU
if defined OPT_SYS if errorlevel !OPT_SYS! goto SYSTEM_MENU_INIT

goto END


:: =========================================================================
:: 3. State Detection Subroutine
:: =========================================================================
:CHECK_STATE
set "SYS_INSTALLED=0"
set "USER_INSTALLED=0"

:: Check for System Mode footprints
schtasks /Query /TN "ErgonomicMouseMapping" >nul 2>&1
if !ERRORLEVEL! equ 0 set "SYS_INSTALLED=1"
if exist "%PUBLIC%\Documents\Scripts\ErgonomicMouse.ahk" set "SYS_INSTALLED=1"

:: Check for User Mode footprints
schtasks /Query /TN "ErgonomicMouseMapping-User" >nul 2>&1
if !ERRORLEVEL! equ 0 set "USER_INSTALLED=1"
if exist "%LOCALAPPDATA%\ErgonomicMouse" set "USER_INSTALLED=1"

:: Determine active mode (mutually exclusive intent)
set "ACTIVE_MODE=NONE"
if "!SYS_INSTALLED!"=="1" if "!USER_INSTALLED!"=="1" (
    set "ACTIVE_MODE=CONFLICT"
)
if "!SYS_INSTALLED!"=="1" if "!USER_INSTALLED!"=="0" set "ACTIVE_MODE=SYSTEM"
if "!USER_INSTALLED!"=="1" if "!SYS_INSTALLED!"=="0" set "ACTIVE_MODE=USER"

exit /b


:: =========================================================================
:: 4. System-Mode Logic
:: =========================================================================
:SYSTEM_MENU_INIT
net session >nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo.
    echo Requesting administrative privileges for System-Mode...
    powershell -Command "Start-Process -FilePath $env:SCRIPT_PATH -ArgumentList '/System' -Verb RunAs"
    exit /b
)

:SYSTEM_MENU
call :CHECK_STATE
cls
echo ===================================================
echo  System-Mode Manager (Elevated)
echo ===================================================
echo.
:: DYNAMIC UX LOCK
if "!USER_INSTALLED!"=="1" (
    echo  [1] Install or Update System Deployment [LOCKED: Remove User-Mode First]
) else (
    echo  [1] Install or Update System Deployment
)
echo  [2] Standard Uninstall (Keep AutoHotkey Engine)
echo  [3] Full Uninstall (Remove Scripts AND Engine)
echo  [4] Back to Main Menu
echo.
choice /C 1234 /N /M "Select an option (1-4): "

if errorlevel 4 goto MAIN_MENU
if errorlevel 3 goto SYS_UNINSTALL_FULL
if errorlevel 2 goto SYS_UNINSTALL_STD
if errorlevel 1 goto SYS_INSTALL


:SYS_INSTALL
:: POLITE INTERCEPTION
if "!USER_INSTALLED!"=="1" (
    cls
    echo ===================================================
    echo  Installation Locked
    echo ===================================================
    echo.
    echo  A User-Mode deployment is currently active on this machine.
    echo  To prevent system conflicts, please completely uninstall the 
    echo  User-Mode version before attempting a System-Mode installation.
    echo.
    pause
    goto SYSTEM_MENU
)

cls
echo ===================================================
echo  Action Selected: Install/Update System-Mode
echo ===================================================
echo.
powershell -NoProfile -ExecutionPolicy Bypass -File "bin\registerErgonomicMouseSchdTask.ps1"
if !ERRORLEVEL! equ 0 (
    echo.
    echo [RESULT] SUCCESS: System deployment completed.
) else (
    echo.
    echo [RESULT] FAILED: Deployment script encountered an error.
)
echo.
powershell Start-Sleep -Seconds 4
goto END


:SYS_UNINSTALL_STD
cls
echo ===================================================
echo  Action Selected: System Standard Uninstall
echo ===================================================
echo.
call :SYS_CLEANUP_SCRIPTS
set "STATUS_ENGINE=SKIPPED (Preserved by choice)"
goto SYS_PRINT_SUMMARY


:SYS_UNINSTALL_FULL
cls
echo ===================================================
echo  Action Selected: System Full Uninstall
echo ===================================================
echo.
call :SYS_CLEANUP_SCRIPTS
echo.
if exist "%ProgramFiles%\AutoHotkey\v2" (
    echo   - AutoHotkey engine found. Removing...
    rmdir /S /Q "%ProgramFiles%\AutoHotkey\v2" >nul 2>&1
    rmdir "%ProgramFiles%\AutoHotkey" >nul 2>&1
    if exist "%ProgramFiles%\AutoHotkey\v2" ( set "STATUS_ENGINE=FAILED (Files may be in use)" ) else ( set "STATUS_ENGINE=SUCCESS" )
) else (
    echo   - No AutoHotkey engine found. Skipping removal.
    set "STATUS_ENGINE=SKIPPED (Not found)"
)
goto SYS_PRINT_SUMMARY


:SYS_PRINT_SUMMARY
echo.
echo ===================================================
echo  System Uninstall Summary Report
echo ===================================================
echo  Processes : !STATUS_PROCESS!
echo  Task      : !STATUS_TASK!
echo  Files     : !STATUS_FILES!
echo  Engine    : !STATUS_ENGINE!
echo ===================================================
set "ANY_FAIL=0"
if "!STATUS_PROCESS!"=="FAILED" set "ANY_FAIL=1"
if "!STATUS_TASK!"=="FAILED" set "ANY_FAIL=1"
if "!STATUS_FILES!"=="FAILED (Some files locked)" set "ANY_FAIL=1"
if "!STATUS_ENGINE!"=="FAILED (Files may be in use)" set "ANY_FAIL=1"
if "!ANY_FAIL!"=="1" (
    echo [RESULT] WARNING: One or more cleanup steps failed.
) else (
    echo [RESULT] SUCCESS: System components were cleanly removed.
)
echo.
pause
goto END


:SYS_CLEANUP_SCRIPTS
echo [State Check] 1. Active AutoHotkey Processes...
tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
if !ERRORLEVEL! equ 0 (
    echo   - Terminating active processes...
    taskkill /F /IM AutoHotkey64.exe >nul 2>&1
    timeout /t 2 /nobreak >nul
    tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
    if !ERRORLEVEL! equ 0 ( set "STATUS_PROCESS=FAILED" ) else ( set "STATUS_PROCESS=SUCCESS" )
) else (
    echo   - No processes found.
    set "STATUS_PROCESS=SKIPPED (Not running)"
)

echo.
echo [State Check] 2. Scheduled Task...
schtasks /Query /TN "ErgonomicMouseMapping" >nul 2>&1
if !ERRORLEVEL! equ 0 (
    echo   - Unregistering task...
    schtasks /Delete /TN "ErgonomicMouseMapping" /F >nul 2>&1
    schtasks /Query /TN "ErgonomicMouseMapping" >nul 2>&1
    if !ERRORLEVEL! equ 0 ( set "STATUS_TASK=FAILED" ) else ( set "STATUS_TASK=SUCCESS" )
) else (
    echo   - Task not found.
    set "STATUS_TASK=SKIPPED (Not found)"
)

echo.
echo [State Check] 3. Application Scripts...
set "FILES_FOUND=0"
set "FILES_FAILED=0"
for %%F in ("ErgonomicMouse.ahk" "LaunchAndUpdate.ps1" "registerErgonomicMouseSchdTask.ps1" "update.log" "ErgonomicMouse.ahk.tmp") do (
    if exist "%PUBLIC%\Documents\Scripts\%%~F" (
        set "FILES_FOUND=1"
        del /Q /F "%PUBLIC%\Documents\Scripts\%%~F" >nul 2>&1
        if exist "%PUBLIC%\Documents\Scripts\%%~F" ( set "FILES_FAILED=1" )
    )
)
if "!FILES_FOUND!"=="1" (
    if "!FILES_FAILED!"=="1" ( set "STATUS_FILES=FAILED (Some files locked)" ) else (
        echo   - Scripts removed successfully.
        set "STATUS_FILES=SUCCESS"
    )
) else (
    echo   - No scripts found.
    set "STATUS_FILES=SKIPPED (None found)"
)
exit /b


:: =========================================================================
:: 5. User-Mode Logic
:: =========================================================================
:USER_MENU
call :CHECK_STATE
cls
echo ===================================================
echo  User-Mode Manager
echo ===================================================
echo.
:: DYNAMIC UX LOCK
if "!SYS_INSTALLED!"=="1" (
    echo  [1] Install or Update User Deployment [LOCKED: Remove System-Mode First]
) else (
    echo  [1] Install or Update User Deployment
)
echo  [2] Uninstall and Clean Up
echo  [3] Back to Main Menu
echo.
choice /C 123 /N /M "Select an option (1-3): "

if errorlevel 3 goto MAIN_MENU
if errorlevel 2 goto USER_UNINSTALL
if errorlevel 1 goto USER_INSTALL


:USER_INSTALL
cls
echo ===================================================
echo  Action Selected: Install/Update User-Mode
echo ===================================================
echo.
set "DEPLOY_SCRIPT=%~dp0bin\registerErgonomicMouseSchdTask-User.ps1"
powershell -NoProfile -ExecutionPolicy Bypass -Command "& $env:DEPLOY_SCRIPT"
if !ERRORLEVEL! equ 0 (
    echo.
    echo [RESULT] SUCCESS: User deployment completed.
) else (
    echo.
    echo [RESULT] FAILED: Deployment script encountered an error.
)
echo.
powershell Start-Sleep -Seconds 4
goto END


:USER_UNINSTALL
cls
echo ===================================================
echo  Action Selected: User Uninstall
echo ===================================================
echo.
echo [State Check] 1. Active AutoHotkey Processes...
tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
if !ERRORLEVEL! equ 0 (
    echo   - Terminating active processes...
    taskkill /F /IM AutoHotkey64.exe >nul 2>&1
    timeout /t 2 /nobreak >nul
    tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
    if !ERRORLEVEL! equ 0 ( set "STATUS_PROCESS=FAILED" ) else ( set "STATUS_PROCESS=SUCCESS" )
) else (
    echo   - No processes found.
    set "STATUS_PROCESS=SKIPPED (Not running)"
)

echo.
echo [State Check] 2. Scheduled Task...
schtasks /Query /TN "ErgonomicMouseMapping-User" >nul 2>&1
if !ERRORLEVEL! equ 0 (
    echo   - Unregistering task...
    schtasks /Delete /TN "ErgonomicMouseMapping-User" /F >nul 2>&1
    schtasks /Query /TN "ErgonomicMouseMapping-User" >nul 2>&1
    if !ERRORLEVEL! equ 0 ( set "STATUS_TASK=FAILED" ) else ( set "STATUS_TASK=SUCCESS" )
) else (
    echo   - Task not found.
    set "STATUS_TASK=SKIPPED (Not found)"
)

echo.
echo [State Check] 3. Application Directory and Engine...
if exist "%LOCALAPPDATA%\ErgonomicMouse" (
    echo   - Application folder found. Removing...
    rmdir /S /Q "%LOCALAPPDATA%\ErgonomicMouse" >nul 2>&1
    if exist "%LOCALAPPDATA%\ErgonomicMouse" ( set "STATUS_FILES=FAILED (Files may be in use)" ) else ( set "STATUS_FILES=SUCCESS" )
) else (
    echo   - Application folder not found.
    set "STATUS_FILES=SKIPPED (Not found)"
)

echo.
echo ===================================================
echo  User Uninstall Summary Report
echo ===================================================
echo  Processes : !STATUS_PROCESS!
echo  Task      : !STATUS_TASK!
echo  App Data  : !STATUS_FILES!
echo ===================================================
set "ANY_FAIL=0"
if "!STATUS_PROCESS!"=="FAILED" set "ANY_FAIL=1"
if "!STATUS_TASK!"=="FAILED" set "ANY_FAIL=1"
if "!STATUS_FILES!"=="FAILED (Files may be in use)" set "ANY_FAIL=1"

if "!ANY_FAIL!"=="1" (
    echo [RESULT] WARNING: One or more cleanup steps failed.
) else (
    echo [RESULT] SUCCESS: User components were cleanly removed.
)
echo.
pause
goto END

:END
exit /b