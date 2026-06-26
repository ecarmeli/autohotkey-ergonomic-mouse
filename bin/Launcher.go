package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// =========================================================================
// 1. Declarations (Constants)
// =========================================================================

const remoteUrl = "https://raw.githubusercontent.com/ecarmeli/autohotkey-ergonomic-mouse/main/bin/ErgonomicMouse.ahk"

// =========================================================================
// 2. Variables (Global State)
// =========================================================================

var (
	targetDir           string
	ahkScriptTargetPath string
	etagFile            string
	ahkExecutable       string
	logFile             string
)

// =========================================================================
// 3. Helper Functions
// =========================================================================

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func logEvent(logPath string, message string) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("%s - %s\n", timestamp, message))
}

func rotateLogs(logFile string) {
	info, err := os.Stat(logFile)
	if err == nil && info.Size() > 1024*1024 { // If > 1MB
		oldLog := logFile + ".old"
		os.Rename(logFile, oldLog)
	}

	oldInfo, err := os.Stat(logFile + ".old")
	if err == nil && time.Since(oldInfo.ModTime()).Hours() > 24*90 { // If > 90 days
		os.Remove(logFile + ".old")
	}
}

func launchAHK(executable string, scriptPath string) {
	if fileExists(executable) && fileExists(scriptPath) {
		// Use .Start() instead of .Run() so the Go program doesn't wait for AHK to close
		cmd := exec.Command(executable, scriptPath)
		cmd.Start()
	}
}

// =========================================================================
// 4. Execution Code
// =========================================================================

func main() {
	// -------------------------------------------------------------------------
	// CLI Flag Parsing & Path Configuration
	// -------------------------------------------------------------------------

	// Define the --mode flag (defaults to "system" if not provided)
	mode := flag.String("mode", "system", "Execution mode: 'system' or 'user'")
	flag.Parse()

	// Route configuration paths based on the flag
	if *mode == "user" {
		targetDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "ErgonomicMouse")
		ahkExecutable = filepath.Join(targetDir, "AutoHotkey", "AutoHotkey64.exe")
	} else {
		targetDir = filepath.Join(os.Getenv("PROGRAMDATA"), "ErgonomicMouse")
		ahkExecutable = filepath.Join(os.Getenv("ProgramFiles"), "AutoHotkey", "v2", "AutoHotkey64.exe")
	}

	// Construct shared paths downstream
	ahkScriptTargetPath = filepath.Join(targetDir, "ErgonomicMouse.ahk")
	etagFile = ahkScriptTargetPath + ".etag"
	logFile = filepath.Join(targetDir, "update.log")

	// -------------------------------------------------------------------------
	// Prepare the Ground
	// -------------------------------------------------------------------------

	// Ensure target directory exists
	os.MkdirAll(targetDir, 0755)

	// Handle Log Rotation
	rotateLogs(logFile)

	// -------------------------------------------------------------------------
	// Conditional HTTP Request (ETag Logic)
	// -------------------------------------------------------------------------

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", remoteUrl, nil)
	if err != nil {
		logEvent(logFile, fmt.Sprintf("WARNING: Failed to build request: %v", err))
		launchAHK(ahkExecutable, ahkScriptTargetPath)
		return
	}

	// If both the local script and ETag file exist, read the ETag and set the header
	if fileExists(ahkScriptTargetPath) && fileExists(etagFile) {
		localETag, err := os.ReadFile(etagFile)
		if err == nil && len(strings.TrimSpace(string(localETag))) > 0 {
			req.Header.Set("If-None-Match", strings.TrimSpace(string(localETag)))
		}
	}

	// Execute the Request
	resp, err := client.Do(req)
	if err != nil {
		logEvent(logFile, fmt.Sprintf("WARNING: Silent update network failure: %v", err))
		launchAHK(ahkExecutable, ahkScriptTargetPath)
		return
	}
	defer resp.Body.Close()

	// -------------------------------------------------------------------------
	// Handle the Response
	// -------------------------------------------------------------------------

	if resp.StatusCode == http.StatusNotModified { // 304 - No changes
		logEvent(logFile, "INFO: Script up to date.")
		launchAHK(ahkExecutable, ahkScriptTargetPath)
		return
	}

	if resp.StatusCode == http.StatusOK { // 200 - New file found
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logEvent(logFile, "WARNING: Failed to read downloaded content.")
			launchAHK(ahkExecutable, ahkScriptTargetPath)
			return
		}

		bodyString := string(bodyBytes)

		// Validation: Check size and presence of the required directive
		if len(bodyString) < 100 || !strings.Contains(strings.ToLower(bodyString), "#requires autohotkey") {
			logEvent(logFile, "WARNING: Downloaded file failed validation checks. Aborting update.")
			launchAHK(ahkExecutable, ahkScriptTargetPath)
			return
		}

		// Atomic File Replacement (Write to .tmp first, then rename)
		tmpScriptPath := ahkScriptTargetPath + ".tmp"
		os.WriteFile(tmpScriptPath, bodyBytes, 0644)
		os.Rename(tmpScriptPath, ahkScriptTargetPath) // Overwrites existing file

		// Save the new ETag
		remoteETag := resp.Header.Get("ETag")
		if remoteETag != "" {
			tmpEtagPath := etagFile + ".tmp"
			os.WriteFile(tmpEtagPath, []byte(remoteETag), 0644)
			os.Rename(tmpEtagPath, etagFile)
		}

		logEvent(logFile, fmt.Sprintf("SUCCESS: Local script updated. ETag: %s", remoteETag))
	} else {
		logEvent(logFile, fmt.Sprintf("WARNING: Unexpected HTTP status: %d", resp.StatusCode))
	}

	// -------------------------------------------------------------------------
	// Launch AutoHotkey
	// -------------------------------------------------------------------------
	launchAHK(ahkExecutable, ahkScriptTargetPath)
}
