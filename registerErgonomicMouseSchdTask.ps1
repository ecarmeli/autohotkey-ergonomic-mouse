# 1. Check for Administrative Privileges
    if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
        Write-Warning "Elevation Required: Please run this script as an Administrator."
        Start-Sleep 2
        Write-Host "Aborting."
        Start-Sleep 1
        Break
    }

# 2. Define Variables
    $TaskName      = "ErgonomicMouseMapping"
    # Ensure this path matches where AutoHotkey v2 is installed on your machine
    $AHKExecutable = "$ENV:ProgramFiles\AutoHotkey\v2\AutoHotkey64.exe" 
    $TargetDir     = "$Env:PUBLIC\Documents\Scripts"
    $ScriptPath    = "$TargetDir\ErgonomicMouse.ahk"
    $SourceAHKfile = "ErgonomicMouse.ahk"                
    $Description   = "Runs the ergonomic mouse remapping script (F5/F6/F7) for all users."

# 3. Prepare Directory and Copy Files
    Write-Host "Ensuring target directory exists: $TargetDir" -ForegroundColor Cyan
    if (-not (Test-Path -Path $TargetDir)) {
        New-Item -ItemType Directory -Path $TargetDir -Force -ErrorAction SilentlyContinue | Out-Null
    }

# Source AHK is assumed to be in the same directory as this running PS script
    $SourceAHKpath = Join-Path -Path $PSScriptRoot -ChildPath $SourceAHKfile -ErrorAction SilentlyContinue

    if (Test-Path -Path $SourceAHKpath) {
        Write-Host "Copying AHK script to Public Documents..."
        Copy-Item -Path $SourceAHKpath -Destination $ScriptPath -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "Could not find 'ErgonomicMouse.ahk' in the same folder as this script ($PSScriptRoot)."
        Start-Sleep 2
        Write-Host "Aborting."
        Start-Sleep 1
        Break
    }

# Copy the deployment script itself
    if ($PSCommandPath) {
        Write-Host "Copying deployment script to Public Documents..."
        Copy-Item -Path $PSCommandPath -Destination $TargetDir -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "Could not determine the path of this PowerShell script (running unsaved?). Skipping PS1 copy."
    }

# 4. Pre-flight File Verification
    $FilesToVerify = @($AHKExecutable, $ScriptPath)
    $MissingFiles = $false

    foreach ($File in $FilesToVerify) {
        if (-not (Test-Path -Path $File)) {
            Write-Error "File not found: $File"
            $MissingFiles = $true
        }
    }

    if ($MissingFiles) {
            Write-Warning "Action required: Ensure the AHK v2 executable is installed at the expected path."
            Start-Sleep 2
            Write-Host "Aborting."
            Start-Sleep 1
            Return
        } else {
            Write-Host "Required files located, continuing with task registration." -ForegroundColor Green -BackgroundColor Black
            Start-Sleep 1
    }

# 5. Define Task Action, Trigger, and Settings
    # Argument is wrapped in quotes to handle any potential spaces in the file path
    $TaskAction = New-ScheduledTaskAction -Execute $AHKExecutable -Argument "`"$ScriptPath`""
    $TaskTrigger = New-ScheduledTaskTrigger -AtLogon
    $TaskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
    $Principal = New-ScheduledTaskPrincipal `
        -GroupId "Users" `
        -RunLevel Highest

# 6. Idempotent Registration (Checks for existing task and updates it)
    $ExistingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($ExistingTask) {
        Write-Host "Updating existing task: '$TaskName'..." -ForegroundColor Cyan
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    }

    Write-Host "Registering task: '$TaskName'..." -ForegroundColor Cyan
    Start-Sleep 1

try {

    # 1. Register/Update the task
        # -Force ensures idempotency by overwriting any existing task with the same name.
        # -ErrorAction Stop is required to redirect errors into the 'catch' block.
        Register-ScheduledTask -Action $TaskAction `
                               -Trigger $TaskTrigger `
                               -Settings $TaskSettings `
                               -TaskName $TaskName `
                               -Description $Description `
                               -Principal $Principal `
                               -Force `
                               -ErrorAction Stop `
                               -Verbose

        Write-Host "Success! '$TaskName' is ready. The tray icon will appear for users at logon." -ForegroundColor Green

    # 2. Immediate Launch (Only runs if the line above succeeded)
        Write-Host "Launching script in current session..." -ForegroundColor Cyan
        Start-ScheduledTask -TaskName $TaskName
        Write-Host "AHK icon should now be visible in your tray." -ForegroundColor Green

    # 3. Get all AHK processes and filter by the command line content
        $SpecificProcess = Get-CimInstance Win32_Process -Filter "Name = 'AutoHotkey64.exe'" |
            Where-Object { $_.CommandLine -like "*$ScriptPath*" }

    # 4. Check the result
        if ($SpecificProcess) {
            Write-Host "Verification Success: '$($SpecificProcess.Name)' is running your script." -ForegroundColor Green
            Write-Host "Process ID: $($SpecificProcess.ProcessId)" -BackgroundColor Blue
            Write-Host "Command Line: $($SpecificProcess.CommandLine)" -BackgroundColor Blue
        } else {
            Write-Warning "Verification Failed: No process is currently running '$ScriptPath'."
        }

    }
    catch {
        Write-Host "Deployment Failed: Could not register the scheduled task '$TaskName'." -ForegroundColor Red
        Write-Host "Technical Details: $($_.Exception)" -ForegroundColor Red
        Start-Sleep 3
        Break
}