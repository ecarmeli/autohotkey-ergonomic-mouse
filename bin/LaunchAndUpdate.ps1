# =========================================================================
# Configuration (Admin Mode)
# =========================================================================
$RemoteUrl           = "https://raw.githubusercontent.com/ecarmeli/autohotkey-ergonomic-mouse/main/bin/ErgonomicMouse.ahk"
$TargetDir           = "$Env:PUBLIC\Documents\Scripts"
$AHKScriptTargetPath = "$TargetDir\ErgonomicMouse.ahk"
$ETagFile            = "$AHKScriptTargetPath.etag"
$AHKExecutable       = "$ENV:ProgramFiles\AutoHotkey\v2\AutoHotkey64.exe"
$LogFile             = "$TargetDir\update.log"

# =========================================================================
# 1. Attempt Silent Update
# =========================================================================

try {

    # ---------------------
    # Prepare the ground:
    # ---------------------
    
    # Ensure the target directory exists
    if (-not (Test-Path -Path $TargetDir)) {
        New-Item -Path $TargetDir -ItemType Directory -Force | Out-Null
    }

    # If the log file exceeds 1MB, archive it to prevent uncontrolled growth
    if ((Test-Path $LogFile) -and ((Get-Item $LogFile).Length -gt 1MB)) {
        Rename-Item $LogFile "$LogFile.old" -Force
    }

    # If the old log file is older than 90 days, delete it to prevent clutter
    if ((Test-Path "$LogFile.old") -and ((Get-Date) - (Get-Item "$LogFile.old").LastWriteTime).Days -gt 90) {
        Remove-Item "$LogFile.old" -Force
    }

    # Ensure TLS 1.2 is enabled
    [Net.ServicePointManager]::SecurityProtocol = `
        [Net.ServicePointManager]::SecurityProtocol -bor `
        [Net.SecurityProtocolType]::Tls12

    # -----------------------------
    # Compare Etag with local files
    # -----------------------------
    
    # Build conditional request headers
    $Headers = @{}

    # Send the ETag to GitHub if BOTH the ETag file AND the AHK script exist locally.
    # If the ETag file exists, read its value and add it to the request headers. We will ask GitHub to return 304 Not Modified if the file hasn't changed since our last download.
    # If the AHK script is missing, this forces a fresh download regardless of the ETag.
    if ((Test-Path $ETagFile) -and (Test-Path $AHKScriptTargetPath)) { 
        $LocalETag = (Get-Content $ETagFile -Raw).Trim()

        if (-not [string]::IsNullOrWhiteSpace($LocalETag)) { 
            $Headers["If-None-Match"] = $LocalETag
        }
    }

    # -----------------------------
    # Download files
    # -----------------------------

    try {
        # Request remote file. 
        # If $Headers contains an ETag, GitHub does the comparison.
        # If $Headers is empty (because files were missing), it downloads instantly.
        $params = @{
            Uri = $RemoteUrl
            Headers = $Headers
            TimeoutSec = 8
            ErrorAction = 'Stop'
        }

        # If running in Windows PowerShell (v5.1 or earlier), use Basic Parsing to avoid issues with the default IE engine.
        if ($PSVersionTable.PSVersion.Major -lt 6) {
            $params.UseBasicParsing = $true
        }

        # Execute the web request with the specified parameters
        $Response = Invoke-WebRequest @params

        # If we reach this line, GitHub sent us new file data (HTTP 200).
        $RemoteETag = $Response.Headers.ETag 

        # Validate that the downloaded content is a valid AHK script by checking file size and for the presence of the "#Requires AutoHotkey" directive.
        # If the file is too small or the directive is missing, throw an error to prevent overwriting the local script with invalid content.
        if (($Response.Content -notmatch "(?i)#Requires AutoHotkey") -or ($Response.Content.Length -lt 100)) {
            throw "Downloaded file does not appear to be a valid AHK script."
        }

        # Write the new file and the new ETag to disk
        [System.IO.File]::WriteAllText("$AHKScriptTargetPath.tmp", $Response.Content, (New-Object System.Text.UTF8Encoding($false))) # Write to a temporary file first to avoid corrupting the existing script if the download fails. Write the new file using native .NET to guarantee a clean UTF-8 output (No Byte Order Mark) 
        Move-Item "$AHKScriptTargetPath.tmp" $AHKScriptTargetPath -Force # Move the temporary file to the final destination, overwriting the old script
        if ($RemoteETag) { # If GitHub provided an ETag, save it to the ETag file for future comparisons
            $RemoteETag | Set-Content -Path "$ETagFile.tmp" -Force
            Move-Item "$ETagFile.tmp" $ETagFile -Force
        }

        "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - SUCCESS: Local script updated. ETag: $RemoteETag" |
            Out-File -FilePath $LogFile -Append -Encoding UTF8
    }
    catch {
        # GitHub returns 304 Not Modified when the ETag matches
        if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq 304) {
            "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - INFO: Script already up to date." |
                Out-File -FilePath $LogFile -Append -Encoding UTF8
        }
        else {
            throw # Not a 304? Throw it to the outer catch block
        }
    }
}
catch {
    "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - WARNING: Silent update failed. Falling back to cached local copy. Technical details: $($_.Exception.ToString())" |
        Out-File -FilePath $LogFile -Append -Encoding UTF8
}

# =========================================================================
# 2. Launch AutoHotkey
# =========================================================================

# The "#SingleInstance Force" directive in the AHK script ensures that only one instance runs at a time.
# If an instance is already running, this command will replace it by launching a new instance.
if ((Test-Path -Path $AHKScriptTargetPath) -and (Test-Path -Path $AHKExecutable)) {
    Start-Process -FilePath $AHKExecutable -ArgumentList "`"$AHKScriptTargetPath`""
}