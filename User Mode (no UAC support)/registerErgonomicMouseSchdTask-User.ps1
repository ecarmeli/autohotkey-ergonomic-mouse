# 1. Define Variables
$TaskName      = "ErgonomicMouseMapping"
$TargetDir     = "$env:LOCALAPPDATA\ErgonomicMouse"
$AHKDir        = "$TargetDir\AutoHotkey"
$AHKExecutable = "$AHKDir\AutoHotkey64.exe" 
$ScriptPath    = "$TargetDir\ErgonomicMouse-User.ahk"
$Description   = "User-scoped ergonomic mouse mapping."

$AHKZipUrl     = "https://www.autohotkey.com/download/ahk-v2.zip"
$ZipPath       = "$TargetDir\ahk-v2.zip"

Write-Host "Starting User-Level Deployment for $env:USERNAME..." -ForegroundColor Cyan

# 2. Prepare Directory
if (-not (Test-Path -Path $TargetDir)) {
    Write-Host "Creating user directory at $TargetDir"
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

# 3. Download and Extract AutoHotkey v2
if (-not (Test-Path -Path $AHKExecutable)) {
    Write-Host "AutoHotkey64.exe not found locally. Downloading from official source..." -ForegroundColor Yellow
    try {

	# Adds TLS 1.2 to the existing supported protocols without overwriting newer ones (like TLS 1.3)
	[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
        
	# -UseBasicParsing ensures compatibility across different PowerShell versions
        Invoke-WebRequest -Uri $AHKZipUrl -OutFile $ZipPath -UseBasicParsing
        
        Write-Host "Extracting AutoHotkey to $AHKDir..." -ForegroundColor Cyan
        Expand-Archive -Path $ZipPath -DestinationPath $AHKDir -Force
        
        # Cleanup the ZIP file
        Remove-Item -Path $ZipPath -Force
        Write-Host "AutoHotkey successfully installed." -ForegroundColor Green
    }
    catch {
        Write-Host "Deployment Failed: Could not download or extract AutoHotkey." -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        Start-Sleep 5
        Break
    }
} else {
    Write-Host "AutoHotkey engine is already present." -ForegroundColor Green
}

# 4. Copy the ErgonomicMouse Script
$SourceAHKFile = Join-Path -Path $PSScriptRoot -ChildPath "ErgonomicMouse-User.ahk"
if (Test-Path -Path $SourceAHKFile) {
    Write-Host "Copying AHK script to LocalAppData..."
    Copy-Item -Path $SourceAHKFile -Destination $ScriptPath -Force
} else {
    Write-Warning "Missing ErgonomicMouse-User.ahk! Ensure it is in the same folder as this deployment script."
    Start-Sleep 5
    Break
}

# 5. Define Task Action and Settings
$TaskAction = New-ScheduledTaskAction -Execute "`"$AHKExecutable`"" -Argument "`"$ScriptPath`""
$TaskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)

# 6. Define User-Specific Trigger and Principal
$TargetUser = "$env:USERDOMAIN\$env:USERNAME"
$TaskTrigger = New-ScheduledTaskTrigger -AtLogon -User $TargetUser
$Principal = New-ScheduledTaskPrincipal -UserId $TargetUser

# 7. Idempotent Registration & Launch
try {
    # Clean up old instances
    Get-CimInstance Win32_Process -Filter "Name = 'AutoHotkey64.exe'" | 
        Where-Object { $_.CommandLine -like "*$ScriptPath*" } | 
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

    Register-ScheduledTask -Action $TaskAction `
                           -Trigger $TaskTrigger `
                           -Settings $TaskSettings `
                           -TaskName $TaskName `
                           -Description $Description `
                           -Principal $Principal `
                           -Force `
                           -ErrorAction Stop

    Write-Host "Success! Task registered for $TargetUser." -ForegroundColor Green
    
    # Launch immediately
    Start-ScheduledTask -TaskName $TaskName
    Start-Sleep 1

    # Verify Execution
    $Verify = Get-CimInstance Win32_Process -Filter "Name = 'AutoHotkey64.exe'" | 
              Where-Object { $_.CommandLine -like "*$ScriptPath*" }

    if ($Verify) {
        Write-Host "Verified: Script is running (PID: $($Verify.ProcessId))." -ForegroundColor Green
    } else {
        Write-Warning "Task registered, but the process verification failed. It may not be running."
    }
}
catch {
    Write-Host "Registration Failed: $($_.Exception.Message)" -ForegroundColor Red
    Start-Sleep 5
}