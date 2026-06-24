# =========================================================================
# Configuration (Admin Mode)
# =========================================================================
$RemoteUrl           = "https://raw.githubusercontent.com/ecarmeli/autohotkey-ergonomic-mouse/main/ErgonomicMouse.ahk"
$TargetDir           = "$Env:PUBLIC\Documents\Scripts"
$AHKScriptTargetPath = "$TargetDir\ErgonomicMouse.ahk"
$AHKExecutable       = "$ENV:ProgramFiles\AutoHotkey\v2\AutoHotkey64.exe"
$TempFile            = "$AHKScriptTargetPath.tmp"
$LogFile             = "$TargetDir\update.log"

# =========================================================================
# 1. Attempt Silent Update
# =========================================================================
try {
    # Ensure TLS 1.2 is enabled for secure HTTPS requests
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    # 1.1. Fetch the expected cryptographic hash from GitHub first
    $ExpectedHash = (Invoke-WebRequest -Uri "$RemoteUrl.sha256" -UseBasicParsing -TimeoutSec 5).Content.Trim()

    # 1.2. Audit local script cache to see if an update is necessary
    if (Test-Path -Path $AHKScriptTargetPath) {
        $CurrentLocalHash = (Get-FileHash -Path $AHKScriptTargetPath -Algorithm SHA256).Hash
        
        # If hashes match perfectly, bypass the network download completely
        if ($CurrentLocalHash -eq $ExpectedHash) {
            $NeedsUpdate = $false
        } else {
            $NeedsUpdate = $true
        }
    }

    # 1.3. Execute the staging and overwrite flow ONLY if a mismatch is detected
    if ($NeedsUpdate) {
        # Download the script to the temporary file path
        Invoke-WebRequest -Uri $RemoteUrl -OutFile $TempFile -UseBasicParsing -TimeoutSec 8

        if (Test-Path -Path $TempFile) {
            # Re-verify the downloaded file's integrity before committing a hot-swap
            $DownloadedHash = (Get-FileHash -Path $TempFile -Algorithm SHA256).Hash

            if ($DownloadedHash -eq $ExpectedHash) {
                Move-Item -Path $TempFile -Destination $AHKScriptTargetPath -Force
                "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - SUCCESS: Local script updated to remote version ($ExpectedHash)." | Out-File -FilePath $LogFile -Append
            } else {
                # Log the corrupted download payload anomaly
                "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - ERROR: Post-download hash mismatch! Expected: $ExpectedHash, Got: $DownloadedHash. Discarding payload." | Out-File -FilePath $LogFile -Append
                Remove-Item -Path $TempFile -Force -ErrorAction SilentlyContinue
            }
        }
    }
}
catch {
    # Fail silently to user, but document the exact network drop details for system debugging
    "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - WARNING: Silent update failed. Falling back to cached local copy. Technical details: $($_.Exception.Message)" | Out-File -FilePath $LogFile -Append
}

# =========================================================================
# 2. Launch AutoHotkey
# =========================================================================
if ((Test-Path -Path $AHKScriptTargetPath) -and (Test-Path -Path $AHKExecutable)) {
    Start-Process -FilePath $AHKExecutable -ArgumentList "`"$AHKScriptTargetPath`""
}