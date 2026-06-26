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
cls
echo ===================================================
echo  Ergonomic Mouse Manager (Unified)
echo ===================================================
echo.

if "!ACTIVE_MODE!"=="SYSTEM" (
    echo  Status: System-Mode is Installed.
) else if "!ACTIVE_MODE!"=="USER" (
    echo  Status: User-Mode is Installed.
) else if "!ACTIVE_MODE!"=="CONFLICT" (
    echo  WARNING: BOTH modes are currently installed.
    echo           Please enter the menus below and uninstall one.
) else (
    echo.
)

if "!IS_ELEVATED!"=="1" ( echo  Privilege: [Administrator] ) else ( echo  Privilege: [Standard User] )

:: Display the elevation warning if applicable
if "!IS_ELEVATED!"=="1" if "!ACTIVE_MODE!" neq "SYSTEM" (
    echo.
    echo  SECURITY LOCK: You are running as Administrator. User-Mode 
    echo  options have been hidden to prevent permission corruption.
    echo  Close this window and run normally to access User-Mode.
)

echo.
echo ===================================================

:: Dynamic menu construction
set "MENU_CHOICES="
set "OPT_SYS="
set "OPT_USER="
set "OPT_EXIT="
set "IDX=1"

:: System Option
if "!ACTIVE_MODE!"=="USER" goto SKIP_SYS
echo  [!IDX!] Manage System-Mode Deployment (Requires Admin)
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_SYS=!IDX!"
set /a IDX+=1
:SKIP_SYS

:: User Option
if "!ACTIVE_MODE!"=="SYSTEM" goto SKIP_USER
if "!IS_ELEVATED!"=="1" goto SKIP_USER
echo  [!IDX!] Manage User-Mode Deployment
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_USER=!IDX!"
set /a IDX+=1
:SKIP_USER

:: Exit Option (Always available)
echo  [!IDX!] Exit
set "MENU_CHOICES=!MENU_CHOICES!!IDX!"
set "OPT_EXIT=!IDX!"

echo.
choice /C !MENU_CHOICES! /N /M "Select an option: "

:: Routing (must check in descending priority order!)
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
set "IS_ELEVATED=0"

:: Check for Elevation
net session >nul 2>&1
if !ERRORLEVEL! equ 0 set "IS_ELEVATED=1"

:: Check for System Mode footprints
schtasks /Query /TN "ErgonomicMouseMapping" >nul 2>&1
if !ERRORLEVEL! equ 0 set "SYS_INSTALLED=1"
if exist "%ProgramData%\ErgonomicMouse\ErgonomicMouse.ahk" set "SYS_INSTALLED=1"

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
echo  [3] Full Uninstall (Remove Application Directory AND Engine)
echo  [4] Back to Main Menu
echo.
choice /C 1234 /N /M "Select an option (1-4): "

if errorlevel 4 goto MAIN_MENU
if errorlevel 3 goto SYS_UNINSTALL_FULL
if errorlevel 2 goto SYS_UNINSTALL_STD
if errorlevel 1 goto SYS_INSTALL


:SYS_INSTALL
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

echo [State Check] 1. Pre-Requisites ^& Environment...
set "STATUS_ENV=SUCCESS"
if exist "%ProgramFiles%\AutoHotkey\v2\AutoHotkey64.exe" (
    echo   - AutoHotkey Engine... Found
) else (
    echo   - Downloading and installing AutoHotkey Engine...
)

:: Execute payload setup silently in background
bin\DeployManager.exe >nul 2>&1
set "EXEC_RESULT=!ERRORLEVEL!"

echo.
echo [State Check] 2. Directory ^& Payload Provisioning...
set "STATUS_FILES=FAILED"
if exist "%ProgramData%\ErgonomicMouse\ErgonomicMouse.ahk" (
    if exist "%ProgramData%\ErgonomicMouse\Launcher.exe" (
        set "STATUS_FILES=SUCCESS"
        echo   - Application payloads... SUCCESS
    )
)
if "!STATUS_FILES!"=="FAILED" echo   - Application payloads... FAILED

echo.
echo [State Check] 3. Windows Scheduled Task...
set "STATUS_TASK=FAILED"
schtasks /Query /TN "ErgonomicMouseMapping" >nul 2>&1
if !ERRORLEVEL! equ 0 (
    if !EXEC_RESULT! equ 0 (
        set "STATUS_TASK=SUCCESS"
        echo   - Registering system service task... SUCCESS
    )
)
if "!STATUS_TASK!"=="FAILED" echo   - Registering system service task... FAILED

echo.
echo ===================================================
echo  System Install Summary Report
echo ===================================================
echo  Environment  : !STATUS_ENV!
echo  Payloads     : !STATUS_FILES!
echo  Service Task : !STATUS_TASK!
echo ===================================================

if "!STATUS_FILES!"=="SUCCESS" if "!STATUS_TASK!"=="SUCCESS" (
    echo.
    echo [RESULT] SUCCESS: System-Mode deployment completed.
    echo          Press Scroll Lock to toggle mappings.
) else (
    echo.
    echo [RESULT] ERROR: One or more deployment steps failed.
)
echo.
pause
goto MAIN_MENU


:SYS_UNINSTALL_STD
cls
echo ===================================================
echo  Action Selected: System Standard Uninstall
echo ===================================================
echo.
call :SYS_CLEANUP_APP
echo.
echo [State Check] 4. AutoHotkey engine...

if exist "%ProgramFiles%\AutoHotkey\v2\AutoHotkey64.exe" (
    echo   - Preserving AutoHotkey engine core by choice.
    set "STATUS_ENGINE=SKIPPED (Preserved)"
) else (
    echo   - AutoHotkey engine core is not found. Skipping removal.
    set "STATUS_ENGINE=SKIPPED (Not found)"
)
goto SYS_PRINT_SUMMARY


:SYS_UNINSTALL_FULL
cls
echo ===================================================
echo  Action Selected: System Full Uninstall
echo ===================================================
echo.
call :SYS_CLEANUP_APP
echo.
echo [State Check] 4. AutoHotkey engine...
if exist "%ProgramFiles%\AutoHotkey\v2" (
    echo   - AutoHotkey engine found. Removing...
    rmdir /S /Q "%ProgramFiles%\AutoHotkey\v2" >nul 2>&1
    rmdir "%ProgramFiles%\AutoHotkey" >nul 2>&1
    if exist "%ProgramFiles%\AutoHotkey\v2" ( set "STATUS_ENGINE=FAILED (Files may be in use)" ) else ( set "STATUS_ENGINE=SUCCESS" )
) else (
    echo   - AutoHotkey engine core is not found. Skipping removal.
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
    echo.
    echo [RESULT] WARNING: One or more cleanup steps failed.
) else (
    echo.
    echo [RESULT] SUCCESS: System components were cleanly removed.
)
echo.
pause
goto MAIN_MENU


:SYS_CLEANUP_APP
echo [State Check] 1. Active AutoHotkey Processes...
tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
if !ERRORLEVEL! equ 0 (
    echo   - Terminating active processes...
    taskkill /F /IM AutoHotkey64.exe >nul 2>&1
    timeout /t 2 /nobreak >nul
    tasklist /FI "IMAGENAME eq AutoHotkey64.exe" 2>NUL | find /I "AutoHotkey64.exe" >NUL
    if !ERRORLEVEL! equ 0 ( set "STATUS_PROCESS=FAILED" ) else ( set "STATUS_PROCESS=SUCCESS" )
) else (
    echo   - No processes found. Skipping termination.
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
    echo   - Task not found. Skipping removal.
    set "STATUS_TASK=SKIPPED (Not found)"
)

echo.
echo [State Check] 3. Application Directory...
if exist "%ProgramData%\ErgonomicMouse\" (
    
    echo   - Removing ErgonomicMouse directory and all contents...
    rmdir /S /Q "%ProgramData%\ErgonomicMouse" >nul 2>&1
    
    if exist "%ProgramData%\ErgonomicMouse\" ( 
        set "STATUS_FILES=FAILED (Folder or files locked)" 
    ) else (
        set "STATUS_FILES=SUCCESS"
    )
) else (
    echo   - Application directory not found. Skipping removal.
    set "STATUS_FILES=SKIPPED (None found)"
)

exit /b


:: =========================================================================
:: 5. User-Mode Logic
:: =========================================================================
:USER_MENU
call :CHECK_STATE

if "!IS_ELEVATED!"=="1" (
    cls
    echo ===================================================
    echo  Access Restricted: Elevation Mismatch
    echo ===================================================
    echo.
    echo  You are currently running this script as an Administrator.
    echo  Managing User-Mode deployments while elevated will corrupt 
    echo  the permissions of your scheduled tasks and folders.
    echo.
    echo  Please exit this window and run the script normally.
    echo.
    pause
    goto MAIN_MENU
)

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
if "!SYS_INSTALLED!"=="1" (
    cls
    echo ===================================================
    echo  Installation Locked
    echo ===================================================
    echo.
    echo  A System-Mode deployment is currently active on this machine.
    echo  To prevent system conflicts, please completely uninstall the 
    echo  System-Mode version before attempting a User-Mode installation.
    echo.
    pause
    goto USER_MENU
)

cls
echo ===================================================
echo  Action Selected: Install/Update User-Mode
echo ===================================================
echo.

echo [State Check] 1. Pre-Requisites ^& Environment...
set "STATUS_ENV=SUCCESS"
if exist "%LOCALAPPDATA%\ErgonomicMouse\AutoHotkey64.exe" (
    echo   - AutoHotkey Engine... Found
) else (
    echo   - Downloading and installing AutoHotkey Engine...
)

:: Execute DeployManager silently in User Mode
bin\DeployManager.exe --mode=user >nul 2>&1
set "EXEC_RESULT=!ERRORLEVEL!"

echo.
echo [State Check] 2. Directory ^& Payload Provisioning...
set "STATUS_FILES=FAILED"
if exist "%LOCALAPPDATA%\ErgonomicMouse" (
    set "STATUS_FILES=SUCCESS"
    echo   - Provisioning sandboxed user directories... SUCCESS
)
if "!STATUS_FILES!"=="FAILED" echo   - Provisioning sandboxed user directories... FAILED

echo.
echo [State Check] 3. Windows Scheduled Task...
set "STATUS_TASK=FAILED"
schtasks /Query /TN "ErgonomicMouseMapping-User" >nul 2>&1
if !ERRORLEVEL! equ 0 (
    if !EXEC_RESULT! equ 0 (
        set "STATUS_TASK=SUCCESS"
        echo   - Registering user service task... SUCCESS
    )
)
if "!STATUS_TASK!"=="FAILED" echo   - Registering user service task... FAILED

echo.
echo ===================================================
echo  User Install Summary Report
echo ===================================================
echo  Environment  : !STATUS_ENV!
echo  Payloads     : !STATUS_FILES!
echo  Service Task : !STATUS_TASK!
echo ===================================================

if "!STATUS_FILES!"=="SUCCESS" if "!STATUS_TASK!"=="SUCCESS" (
    echo.
    echo [RESULT] SUCCESS: User-Mode deployment completed.
    echo          Press Scroll Lock to toggle mappings.
) else (
    echo.
    echo [RESULT] ERROR: One or more deployment steps failed.
)
echo.
pause
goto MAIN_MENU


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
    echo   - No processes found. Skipping termination.
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
    echo   - Task not found. Skipping removal.
    set "STATUS_TASK=SKIPPED (Not found)"
)

echo.
echo [State Check] 3. Application Directory and Engine...
if exist "%LOCALAPPDATA%\ErgonomicMouse" (
    echo   - Application folder found. Removing...
    rmdir /S /Q "%LOCALAPPDATA%\ErgonomicMouse" >nul 2>&1
    if exist "%LOCALAPPDATA%\ErgonomicMouse" ( set "STATUS_FILES=FAILED (Files may be in use)" ) else ( set "STATUS_FILES=SUCCESS" )
) else (
    echo   - Application folder not found. Skipping removal.
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
    echo.
    echo [RESULT] WARNING: One or more cleanup steps failed.
) else (
    echo.
    echo [RESULT] SUCCESS: User components were cleanly removed.
)
echo.
pause
goto MAIN_MENU

:END
exit /b