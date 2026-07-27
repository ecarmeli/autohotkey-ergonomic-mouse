package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "none"
)

// =========================================================================
// 1. Constants
// =========================================================================

// GitHub API URL for checking the latest published release
const releasesAPI = "https://api.github.com/repos/ecarmeli/autohotkey-ergonomic-mouse/releases/latest"

// =========================================================================
// 2. Configuration Struct
// =========================================================================

type Config struct {
	Mode          string
	TargetDir     string
	AHKScriptPath string
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

	// 3. Map runtime assets relative to the installation directory
	cfg.AHKExe = filepath.Join(cfg.TargetDir, "AutoHotkey", "AutoHotkey64.exe")
	cfg.AHKScriptPath = filepath.Join(cfg.TargetDir, "ErgonomicMouse.ahk")

	// 4. Store logs outside the installation/program directory.
	// In system mode, the launcher runs from ProgramData, but logs should
	// belong to the interactive user's LocalAppData profile.
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		cfg.LogFile = filepath.Join(localAppData, "ErgonomicMouse", "logs", "launcher.log")
	} else {
		cfg.LogFile = filepath.Join(cfg.TargetDir, "logs", "launcher.log")
	}

	return cfg, nil
}

// =========================================================================
// 3. Helper Functions
// =========================================================================

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false // Safely handle permission denials or missing files without panicking
	}
	return !info.IsDir()
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

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	return value
}

func parseVersion(value string) ([3]int, error) {
	var result [3]int

	normalized := normalizeVersion(value)
	parts := strings.Split(normalized, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("unsupported version format: %s", value)
	}

	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return result, fmt.Errorf("invalid version component %q in %s: %w", part, value, err)
		}
		result[i] = number
	}

	return result, nil
}

func isNewerVersion(latest string, current string) (bool, error) {
	latestVersion, err := parseVersion(latest)
	if err != nil {
		return false, err
	}

	currentVersion, err := parseVersion(current)
	if err != nil {
		return false, err
	}

	for i := 0; i < 3; i++ {
		if latestVersion[i] > currentVersion[i] {
			return true, nil
		}
		if latestVersion[i] < currentVersion[i] {
			return false, nil
		}
	}

	return false, nil
}

// =========================================================================
// 4. Execution Code
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
	// Check for Updates (Notification Only)
	// -------------------------------------------------------------------------

	// Skip the version check if this is a local development build
	if version != "dev" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", releasesAPI, nil)
		if err != nil {
			logToFile(cfg.LogFile, "INFO: Failed to build update check request: %v", err)
		} else {
			req.Header.Set("Accept", "application/vnd.github.v3+json")
			req.Header.Set("User-Agent", "ErgonomicMouseKeys/"+version)

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				logToFile(cfg.LogFile, "INFO: Update check failed: %v", err)
			} else {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					var release struct {
						TagName string `json:"tag_name"`
						HTMLURL string `json:"html_url"`
						Assets  []struct {
							Name               string `json:"name"`
							BrowserDownloadURL string `json:"browser_download_url"`
						} `json:"assets"`
					}

					if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
						logToFile(cfg.LogFile, "INFO: Failed to parse update check response: %v", err)
					} else {
						updateAvailable, err := isNewerVersion(release.TagName, version)
						if err != nil {
							logToFile(cfg.LogFile, "INFO: Unable to compare versions. current=%s latest=%s error=%v", version, release.TagName, err)
						} else if updateAvailable {
							var directDownloadURL string
							for _, asset := range release.Assets {
								if asset.Name == "ErgonomicMouseSetup.exe" {
									directDownloadURL = asset.BrowserDownloadURL
									break
								}
							}

							logToFile(cfg.LogFile, "INFO: Update available. current=%s latest=%s", version, release.TagName)
							logToFile(cfg.LogFile, "INFO: Release page: %s", release.HTMLURL)

							if directDownloadURL != "" {
								logToFile(cfg.LogFile, "INFO: Direct installer download: %s", directDownloadURL)
							}
						} else {
							if normalizeVersion(release.TagName) == normalizeVersion(version) {
								logToFile(cfg.LogFile, "INFO: Running the latest published version (%s).", version)
							} else {
								logToFile(cfg.LogFile, "INFO: No newer release found. current=%s latest=%s", version, release.TagName)
							}
						}
					}
				} else {
					logToFile(cfg.LogFile, "INFO: Update check returned HTTP status: %d", resp.StatusCode)
				}
			}
		}
	}

	// -------------------------------------------------------------------------
	// Launch Engine
	// -------------------------------------------------------------------------

	// Final launch of the AutoHotkey script
	launchAHK(cfg)

	logToFile(cfg.LogFile, "Launcher execution completed.")

}
