# 1. Define Variables
    $TaskName             = "ErgonomicMouseMapping-User"
    $TargetDir            = "$env:LOCALAPPDATA\ErgonomicMouse"
    $AHKDir               = "$TargetDir\AutoHotkey"
    $AHKExecutable        = "$AHKDir\AutoHotkey64.exe"
    $AHKScriptFileName    = "ErgonomicMouse-User.ahk"
    $AHKScriptTargetPath  = "$TargetDir\$AHKScriptFileName"
    $LauncherFileName     = "LaunchAndUpdate-User.ps1"
    $LauncherTargetPath   = "$TargetDir\$LauncherFileName"
    $Description          = "Runs the ergonomic mouse remapping script (F5/F6/F7) for the current user."

    $AHKZipUrl            = "https://www.autohotkey.com/download/ahk-v2.zip"
    $ZipPath              = "$TargetDir\ahk-v2.zip"

    Write-Host "Starting User-Level Deployment for $env:USERNAME..." -ForegroundColor Cyan

# 2. Prepare Directory and Download Executable
    if (-not (Test-Path -Path $TargetDir)) {
        Write-Host "Creating user directory at $TargetDir..." -ForegroundColor Cyan
        New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
    }

    if (-not (Test-Path -Path $AHKExecutable)) {
        Write-Host "AutoHotkey64.exe not found locally. Downloading from official source..." -ForegroundColor Yellow
        try {
            # Adds TLS 1.2 to the existing supported protocols without overwriting newer ones (like TLS 1.3)
            [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
            
            # -UseBasicParsing ensures compatibility across different PowerShell versions
            Invoke-WebRequest -Uri $AHKZipUrl -OutFile $ZipPath -UseBasicParsing
            
            Write-Host "Extracting AutoHotkey to $AHKDir..." -ForegroundColor Cyan
            Expand-Archive -Path $ZipPath -DestinationPath $AHKDir -Force
            
            # Clean up the ZIP file
            Remove-Item -Path $ZipPath -Force
            Write-Host "AutoHotkey successfully installed." -ForegroundColor Green
        }
        catch {
            Write-Host "Deployment Failed: Could not download or extract AutoHotkey engine." -ForegroundColor Red
            Write-Host "Technical Details: $($_.Exception.Message)" -ForegroundColor Red
            Start-Sleep 3
            Return
        }
    } else {
        Write-Host "AutoHotkey engine is already installed locally." -ForegroundColor Green
    }

# 3. Pre-flight Engine Verification
    # Check if the AutoHotkey executable exists at the expected path
    if (-not (Test-Path -Path $AHKExecutable)) {
        Write-Error "Required AutoHotkey v2 engine not found at: $AHKExecutable"
        Write-Warning "Action required: Automated installer failed to provision AutoHotkey core binary."
        Start-Sleep 2 ; Write-Host "Aborting." ; Start-Sleep 1 ; Return
    }

# 4. Copy Files
    # Copy files safely (Verification is handled implicitly by ensuring source files exist)
    $SourceAHKpath = Join-Path -Path $PSScriptRoot -ChildPath $AHKScriptFileName -ErrorAction SilentlyContinue
    if (Test-Path -Path $SourceAHKpath) { # Copy the AHK script to the target directory
        Write-Host "Copying AHK script to LocalAppData..."
        Copy-Item -Path $SourceAHKpath -Destination $AHKScriptTargetPath -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "Could not find '$AHKScriptFileName' in the source folder."
        Start-Sleep 2 ; Write-Host "Aborting." ; Start-Sleep 1 ; Return
    }

    $LauncherFileNamePath = Join-Path -Path $PSScriptRoot -ChildPath $LauncherFileName -ErrorAction SilentlyContinue
    if (Test-Path -Path $LauncherFileNamePath) { # Copy the launcher script to the target directory
        Write-Host "Copying auto-updater engine to LocalAppData..."
        Copy-Item -Path $LauncherFileNamePath -Destination $LauncherTargetPath -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "Could not find '$LauncherFileName' in the source folder."
        Start-Sleep 2 ; Write-Host "Aborting." ; Start-Sleep 1 ; Return
    }

    if ($PSCommandPath) { # Copy the deployment script itself to the target directory for future reference
        Copy-Item -Path $PSCommandPath -Destination $TargetDir -Force -ErrorAction SilentlyContinue
    } else {
        # This step will fail if the script is run from an unsaved state in the ISE or VSCode, but it's not critical for functionality
        Write-Warning "Could not find the deployment script in the source folder. Copy the file manually to the target directory."
    }

# 5. Define Task Action, Trigger, and Settings
    # Argument is wrapped in quotes to handle any potential spaces in the file path
    # Task targets PowerShell running hidden to process the update engine cleanly
    $TaskAction   = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$LauncherTargetPath`""
    $TargetUser   = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    $TaskTrigger  = New-ScheduledTaskTrigger -AtLogon -User $TargetUser
    $TaskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
    $Principal    = New-ScheduledTaskPrincipal -UserId $TargetUser

# 6. Idempotent Registration
    Write-Host "Registering task: '$TaskName'..." -ForegroundColor Cyan
    Start-Sleep 1

try {
    # 6.1. Clean up old process instances matching this specific script path before renewing configuration
    Get-CimInstance Win32_Process -Filter "Name = 'AutoHotkey64.exe'" | 
        Where-Object { $_.CommandLine -like "*$AHKScriptTargetPath*" } | 
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

    # 6.2. Register/Update the task
        # -Force ensures idempotency by overwriting any existing task with the same name.
        # -ErrorAction Stop is required to redirect errors into the 'catch' block.
        Register-ScheduledTask -Action $TaskAction `
                               -Trigger $TaskTrigger `
                               -Settings $TaskSettings `
                               -TaskName $TaskName `
                               -Description $Description `
                               -Principal $Principal `
                               -Force `
                               -ErrorAction Stop

        Write-Host "Success! '$TaskName' is ready. The tray icon will appear for users at logon." -ForegroundColor Green

    # 6.3. Immediate Launch (Only runs if the line above succeeded)
        Write-Host "Launching script in current session..." -ForegroundColor Cyan
        Start-ScheduledTask -TaskName $TaskName

    # 6.4. Active Polling Verification Loop
        Write-Host "Waiting for update engine initialization and process startup..." -ForegroundColor Gray
        $MaxTimeoutSec = 15
        $ElapsedTime   = 0
        $SpecificProcess = $null

        while ($ElapsedTime -lt $MaxTimeoutSec) {
            # Check if the AHK process running our specific script path exists yet
            $SpecificProcess = Get-CimInstance Win32_Process -Filter "Name = 'AutoHotkey64.exe'" |
                Where-Object { $_.CommandLine -like "*$AHKScriptTargetPath*" }
            
            # If found, break out of the loop immediately to execute verification summary
            if ($SpecificProcess) { break }
            
            # Otherwise, wait 1 second and increment counter
            Start-Sleep -Seconds 1
            $ElapsedTime++
        }

    # 6.5. Evaluate and display verification results
        if ($SpecificProcess) {
            Write-Host "Verification Success: '$($SpecificProcess.Name)' is running your script." -ForegroundColor Green
            Write-Host "Process ID: $($SpecificProcess.ProcessId)" -BackgroundColor Blue
            Write-Host "Command Line: $($SpecificProcess.CommandLine)" -BackgroundColor Blue
        } else {
            Write-Warning "Verification Timeout: Task was triggered, but 'AutoHotkey64.exe' running '$AHKScriptTargetPath' did not initialize within $MaxTimeoutSec seconds."
        }
    }
    catch {
        Write-Host "Deployment Failed: Could not register the scheduled task '$TaskName'." -ForegroundColor Red
        Write-Host "Technical Details: $($_.Exception)" -ForegroundColor Red
        Start-Sleep 3
        Return
}