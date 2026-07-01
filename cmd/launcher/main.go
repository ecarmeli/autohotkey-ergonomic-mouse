package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	version              = "dev"
	buildTime            = "unknown"
	gitCommit            = "none"
	remoteScriptURL      = ""
	expectedScriptSHA256 = ""
)

// =========================================================================
// 1. Configuration Struct
// =========================================================================

type Config struct {
	Mode          string
	TargetDir     string
	AHKScriptPath string
	ETagFile      string
	AHKExe        string
	LogFile       string
}

func buildConfig() (*Config, error) {
	mode := flag.String("mode", "system", "Execution mode: 'system' or 'user'")
	flag.Parse()

	if *mode != "system" && *mode != "user" {
		return nil, fmt.Errorf("invalid mode %q: expected 'system' or 'user'", *mode)
	}

	cfg := &Config{Mode: *mode}

	// 1. Get the absolute path of wherever Launcher.exe is currently running from
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("could not determine execution path: %v", err)
	}

	// 2. The TargetDir is simply the folder containing Launcher.exe
	cfg.TargetDir = filepath.Dir(exePath)

	// 3. Map all internal assets relative to that exact directory
	cfg.AHKExe = filepath.Join(cfg.TargetDir, "AutoHotkey", "AutoHotkey64.exe")
	cfg.AHKScriptPath = filepath.Join(cfg.TargetDir, "ErgonomicMouse.ahk")
	cfg.ETagFile = cfg.AHKScriptPath + ".etag"
	cfg.LogFile = filepath.Join(cfg.TargetDir, "logs", "launcher.log")

	return cfg, nil
}

// =========================================================================
// 2. Helper Functions
// =========================================================================

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false // Safely handle permission denials or missing files without panicking
	}
	return !info.IsDir()
}

func readLimitedBody(r io.Reader, maxSize int64) ([]byte, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return nil, err
	}

	if int64(len(bodyBytes)) > maxSize {
		return nil, fmt.Errorf("downloaded content exceeds maximum allowed size of %d bytes", maxSize)
	}

	return bodyBytes, nil
}

func verifySHA256(data []byte, expectedHash string) error {
	sum := sha256.Sum256(data)
	actualHash := hex.EncodeToString(sum[:])

	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func rotateLogs(logFile string) {
	info, err := os.Stat(logFile)
	if err == nil && info.Size() > 1024*1024 {
		oldLog := logFile + ".old"
		_ = os.Rename(logFile, oldLog)
	}

	oldInfo, err := os.Stat(logFile + ".old")
	if err == nil && time.Since(oldInfo.ModTime()).Hours() > 24*90 {
		_ = os.Remove(logFile + ".old")
	}
}

func launchAHK(cfg *Config) {
	executable := cfg.AHKExe
	scriptPath := cfg.AHKScriptPath

	if fileExists(executable) && fileExists(scriptPath) {
		cmd := exec.Command(executable, scriptPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			logToFile(cfg.LogFile, "ERROR: Failed to launch AutoHotkey: %v", err)
		} else {
			logToFile(cfg.LogFile, "SUCCESS: AutoHotkey launched successfully.")
		}
	} else {
		logToFile(cfg.LogFile, "ERROR: Cannot launch. Executable or script missing.")
	}
}

func logToFile(logPath string, format string, args ...any) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	f.WriteString(fmt.Sprintf("%s - %s\n", ts, msg))
}

// =========================================================================
// 3. Execution Code
// =========================================================================

func main() {
	cfg, err := buildConfig()
	if err != nil {
		log.Fatalf("Fatal: Failed to build configuration: %v", err)
	}

	if err := os.MkdirAll(cfg.TargetDir, 0755); err != nil {
		log.Fatalf("Fatal: Could not create target directory: %v", err)
	}

	// Create the logs subdirectory safely
	if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0755); err != nil {
		log.Fatalf("Fatal: Could not create logs directory: %v", err)
	}

	rotateLogs(cfg.LogFile)

	logToFile(cfg.LogFile, "--- Launcher Started (Mode: %s) ---", cfg.Mode)
	logToFile(cfg.LogFile, "Launcher Version: %s | Build: %s | Commit: %s", version, buildTime, gitCommit)

	if !fileExists(cfg.AHKExe) {
		logToFile(cfg.LogFile, "ERROR: AutoHotkey executable not found at: %s. Aborting.", cfg.AHKExe)
		return
	}

	// -------------------------------------------------------------------------
	// Context-Aware HTTP Request
	// -------------------------------------------------------------------------
	if strings.TrimSpace(remoteScriptURL) == "" || strings.TrimSpace(expectedScriptSHA256) == "" {
		logToFile(cfg.LogFile, "WARNING: Update metadata missing. Skipping remote update.")
		launchAHK(cfg)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", remoteScriptURL, nil)
	if err != nil {
		logToFile(cfg.LogFile, "WARNING: Failed to build request: %v", err)
		launchAHK(cfg)
		return
	}

	if fileExists(cfg.AHKScriptPath) && fileExists(cfg.ETagFile) {
		localETag, err := os.ReadFile(cfg.ETagFile)
		if err == nil && len(strings.TrimSpace(string(localETag))) > 0 {
			req.Header.Set("If-None-Match", strings.TrimSpace(string(localETag)))
		}
	}

	// Enforce hard timeout to prevent socket hangs
	client := &http.Client{Timeout: 15 * time.Second}

	var resp *http.Response
	var doErr error

	// Quick retry logic for minor network blips
	for i := 0; i < 2; i++ {
		resp, doErr = client.Do(req)
		if doErr == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if doErr != nil {
		logToFile(cfg.LogFile, "WARNING: Silent update network failure after retries: %v", doErr)
		launchAHK(cfg)
		return
	}
	defer resp.Body.Close()

	// -------------------------------------------------------------------------
	// Handle the Response Safely
	// -------------------------------------------------------------------------
	if resp.StatusCode == http.StatusNotModified { // 304 - No changes
		logToFile(cfg.LogFile, "INFO: Script up to date (304).")
		launchAHK(cfg)
		return
	}

	if resp.StatusCode == http.StatusOK { // 200 - New file found
		// Lightweight integrity check
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text") && !strings.Contains(contentType, "plain") {
			logToFile(cfg.LogFile, "WARNING: Unexpected content-type: %s. Aborting update.", contentType)
			launchAHK(cfg)
			return
		}

		// Security: Prevent memory exhaustion and reject oversized downloads.
		const maxScriptSize int64 = 5 * 1024 * 1024

		bodyBytes, err := readLimitedBody(resp.Body, maxScriptSize)
		if err != nil {
			logToFile(cfg.LogFile, "WARNING: Failed to read downloaded content safely: %v", err)
			launchAHK(cfg)
			return
		}

		// Security: Verify the downloaded script is exactly the expected build-pinned artifact.
		if err := verifySHA256(bodyBytes, expectedScriptSHA256); err != nil {
			logToFile(cfg.LogFile, "WARNING: Downloaded script failed SHA256 verification: %v", err)
			launchAHK(cfg)
			return
		}

		bodyString := string(bodyBytes)

		// Hardened Script Validation
		if len(bodyBytes) < 100 ||
			!strings.Contains(strings.ToLower(bodyString), "#requires autohotkey") ||
			!strings.Contains(bodyString, "::") {
			logToFile(cfg.LogFile, "WARNING: Downloaded file failed rigorous validation checks. Aborting update.")
			launchAHK(cfg)
			return
		}

		// Hardened Atomic File Replacement
		tmpScriptPath := cfg.AHKScriptPath + ".tmp"
		if err := os.WriteFile(tmpScriptPath, bodyBytes, 0644); err != nil {
			logToFile(cfg.LogFile, "WARNING: Failed to write temp script file: %v", err)
			launchAHK(cfg)
			return
		}

		if err := os.Rename(tmpScriptPath, cfg.AHKScriptPath); err != nil {
			logToFile(cfg.LogFile, "WARNING: Failed to atomically replace script file: %v", err)
			os.Remove(tmpScriptPath) // Clean up orphan temp file
			launchAHK(cfg)
			return
		}

		remoteETag := resp.Header.Get("ETag")
		if remoteETag != "" {
			tmpEtagPath := cfg.ETagFile + ".tmp"
			if err := os.WriteFile(tmpEtagPath, []byte(remoteETag), 0644); err != nil {
				logToFile(cfg.LogFile, "WARNING: Failed to write temp ETag file: %v", err)
			} else {
				if err := os.Rename(tmpEtagPath, cfg.ETagFile); err != nil {
					logToFile(cfg.LogFile, "WARNING: Failed to atomically save ETag file: %v", err)
				}
			}
		} else {
			logToFile(cfg.LogFile, "INFO: No ETag returned by the remote server.")
		}

		logToFile(cfg.LogFile, "SUCCESS: Local script updated. ETag: %s", remoteETag)
	} else {
		logToFile(cfg.LogFile, "WARNING: Unexpected HTTP status: %d", resp.StatusCode)
	}

	// -------------------------------------------------------------------------
	// Launch Engine
	// -------------------------------------------------------------------------

	// Final launch of the AutoHotkey script
	launchAHK(cfg)

	logToFile(cfg.LogFile, "Launcher execution completed.")

}
