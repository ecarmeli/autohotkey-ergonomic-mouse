package main

import (
	"archive/zip"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// =========================================================================
// 1. Constants (Replacing Magic Numbers)
// =========================================================================

const (
	ahkZipURL = "https://www.autohotkey.com/download/ahk-v2.zip"

	// Windows Task Scheduler COM Enums
	TaskActionExec       = 0
	TaskTriggerLogon     = 9
	TaskCreateOrUpdate   = 6
	TaskLogonInteractive = 3
	TaskLogonNone        = 0
	TaskRunLevelHighest  = 1
)

// =========================================================================
// 2. Configuration Struct (Eliminating Global State Mutation)
// =========================================================================

// Config holds all deployment paths and metadata
type Config struct {
	Mode         string
	TargetDir    string
	AHKDir       string
	AHKExe       string
	ScriptDest   string
	LauncherDest string
	ZipPath      string
	ExeDir       string
	LogPath      string
	TaskName     string
	Description  string
	UserContext  string
	IsSystem     bool
	LauncherArgs string
}

// buildConfig parses flags and constructs the immutable configuration state
func buildConfig() (*Config, error) {
	mode := flag.String("mode", "system", "Installation mode: 'system' or 'user'")
	flag.Parse()

	if *mode != "system" && *mode != "user" {
		return nil, fmt.Errorf("invalid mode %q: expected 'system' or 'user'", *mode)
	}

	exePath, err := os.Executable()
	exeDir := "."
	if err == nil {
		exeDir = filepath.Dir(exePath)
	}

	cfg := &Config{
		Mode:         *mode,
		ExeDir:       exeDir,
		LauncherArgs: fmt.Sprintf("--mode=%s", *mode),
	}

	if cfg.Mode == "user" {
		localAppData, err := requiredEnv("LOCALAPPDATA")
		if err != nil {
			return nil, err
		}

		user, err := requiredEnv("USERNAME")
		if err != nil {
			return nil, err
		}

		cfg.IsSystem = false
		cfg.TargetDir = filepath.Join(localAppData, "ErgonomicMouse")
		cfg.AHKDir = filepath.Join(cfg.TargetDir, "AutoHotkey")
		cfg.TaskName = "ErgonomicMouseMapping-User"
		cfg.Description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for the current user."

		domain := os.Getenv("USERDOMAIN")
		if domain != "" {
			cfg.UserContext = domain + "\\" + user
		} else {
			cfg.UserContext = user
		}
	} else {
		programData, err := requiredEnv("ProgramData")
		if err != nil {
			return nil, err
		}

		programFiles, err := requiredEnv("ProgramFiles")
		if err != nil {
			return nil, err
		}

		cfg.IsSystem = true
		cfg.TargetDir = filepath.Join(programData, "ErgonomicMouse")
		cfg.AHKDir = filepath.Join(programFiles, "AutoHotkey", "v2")
		cfg.TaskName = "ErgonomicMouseMapping"
		cfg.Description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for all users."
	}

	cfg.AHKExe = filepath.Join(cfg.AHKDir, "AutoHotkey64.exe")
	cfg.ScriptDest = filepath.Join(cfg.TargetDir, "ErgonomicMouse.ahk")
	cfg.LauncherDest = filepath.Join(cfg.TargetDir, "Launcher.exe")
	cfg.ZipPath = filepath.Join(cfg.TargetDir, "ahk-v2.zip")
	cfg.LogPath = filepath.Join(cfg.TargetDir, "logs", "install.log")

	return cfg, nil
}

// =========================================================================
// 3. Helper Functions
// =========================================================================

// requiredEnv retrieves the value of an environment variable and returns an error if it's not set
func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

// isAdmin checks if the current process has administrative privileges
func isAdmin() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation uint32
	var outLen uint32

	err = windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&outLen,
	)
	if err != nil {
		return false
	}

	return elevation != 0
}

// fileExists checks if a given file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// setupFileLogging configures logging to both stdout and a log file in the target directory
func setupFileLogging(logPath string) *os.File {
	logDir := filepath.Dir(logPath)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.SetOutput(os.Stdout)
		log.Printf("Warning: could not create log directory %s: %v", logDir, err)
		return nil
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		log.SetOutput(os.Stdout)
		log.Printf("Warning: could not open log file %s: %v", logPath, err)
	}

	return file
}

// copyFile copies a file from src to dst. It returns an error if any operation fails.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// downloadFile downloads a file from the specified URL and saves it to the finalPath.
// It uses a temporary file to ensure atomicity and cleans up on failure.
func downloadFile(url string, finalPath string) error {
	client := &http.Client{Timeout: 60 * time.Second}

	tmpPath := finalPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	resp, err := client.Get(url)
	if err != nil {
		out.Close()
		os.Remove(tmpPath)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()

	if copyErr != nil {
		os.Remove(tmpPath)
		return copyErr
	}

	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}

	return os.Rename(tmpPath, finalPath)
}

// unzip extracts a zip archive to the specified destination directory, ensuring no path traversal occurs.
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	destClean := filepath.Clean(destAbs) + string(os.PathSeparator)

	for _, f := range r.File {
		fpath := filepath.Clean(filepath.Join(destAbs, f.Name))

		if !strings.HasPrefix(fpath, destClean) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		closeOutErr := outFile.Close()
		closeReadErr := rc.Close()

		if copyErr != nil {
			return copyErr
		}

		if closeOutErr != nil {
			return closeOutErr
		}

		if closeReadErr != nil {
			return closeReadErr
		}
	}

	return nil
}

// =========================================================================
// 4. Unified COM Task Registration
// =========================================================================

func putCOMProperty(dispatch *ole.IDispatch, name string, value interface{}) error {
	if _, err := oleutil.PutProperty(dispatch, name, value); err != nil {
		return fmt.Errorf("failed to set COM property %s: %w", name, err)
	}
	return nil
}

// registerTaskWithCOM registers a scheduled task using Windows COM API, handling both system and user modes.
func registerTaskWithCOM(cfg *Config) error {
	err := ole.CoInitialize(0)
	if err != nil {
		return err
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Schedule.Service")
	if err != nil {
		return err
	}
	defer unknown.Release()

	service, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer service.Release()

	if _, err = oleutil.CallMethod(service, "Connect"); err != nil {
		return err
	}

	rootFolderProg, err := oleutil.CallMethod(service, "GetFolder", "\\")
	if err != nil {
		return err
	}
	rootFolder := rootFolderProg.ToIDispatch()
	defer rootFolder.Release()

	// Ignore "task not found" here; RegisterTaskDefinition below will create/update it.
	oleutil.CallMethod(rootFolder, "DeleteTask", cfg.TaskName, 0)

	newTaskSettingProg, err := oleutil.CallMethod(service, "NewTask", 0)
	if err != nil {
		return err
	}
	taskDefinition := newTaskSettingProg.ToIDispatch()
	defer taskDefinition.Release()

	regInfoProg, err := oleutil.GetProperty(taskDefinition, "RegistrationInfo")
	if err != nil {
		return err
	}
	regInfo := regInfoProg.ToIDispatch()
	if err := putCOMProperty(regInfo, "Description", cfg.Description); err != nil {
		regInfo.Release()
		return err
	}
	regInfo.Release()

	principalProg, err := oleutil.GetProperty(taskDefinition, "Principal")
	if err != nil {
		return err
	}
	principal := principalProg.ToIDispatch()

	if cfg.IsSystem {
		if err := putCOMProperty(principal, "GroupId", "Builtin\\Users"); err != nil {
			principal.Release()
			return err
		}

		if err := putCOMProperty(principal, "RunLevel", TaskRunLevelHighest); err != nil {
			principal.Release()
			return err
		}
	} else {
		if err := putCOMProperty(principal, "UserId", cfg.UserContext); err != nil {
			principal.Release()
			return err
		}

		if err := putCOMProperty(principal, "LogonType", TaskLogonInteractive); err != nil {
			principal.Release()
			return err
		}
	}
	principal.Release()

	settingsProg, err := oleutil.GetProperty(taskDefinition, "Settings")
	if err != nil {
		return err
	}
	settings := settingsProg.ToIDispatch()

	if err := putCOMProperty(settings, "DisallowStartIfOnBatteries", false); err != nil {
		settings.Release()
		return err
	}
	if err := putCOMProperty(settings, "StopIfGoingOnBatteries", false); err != nil {
		settings.Release()
		return err
	}
	if err := putCOMProperty(settings, "ExecutionTimeLimit", "PT0S"); err != nil {
		settings.Release()
		return err
	}
	settings.Release()

	triggersProg, err := oleutil.GetProperty(taskDefinition, "Triggers")
	if err != nil {
		return err
	}
	triggers := triggersProg.ToIDispatch()

	triggerProg, err := oleutil.CallMethod(triggers, "Create", TaskTriggerLogon)
	if err != nil {
		triggers.Release()
		return err
	}
	trigger := triggerProg.ToIDispatch()

	if !cfg.IsSystem {
		if err := putCOMProperty(trigger, "UserId", cfg.UserContext); err != nil {
			trigger.Release()
			triggers.Release()
			return err
		}
	}

	trigger.Release()
	triggers.Release()

	actionsProg, err := oleutil.GetProperty(taskDefinition, "Actions")
	if err != nil {
		return err
	}
	actions := actionsProg.ToIDispatch()

	actionNewProg, err := oleutil.CallMethod(actions, "Create", TaskActionExec)
	if err != nil {
		actions.Release()
		return err
	}
	action := actionNewProg.ToIDispatch()

	if err := putCOMProperty(action, "Path", cfg.LauncherDest); err != nil {
		action.Release()
		actions.Release()
		return err
	}

	if err := putCOMProperty(action, "Arguments", cfg.LauncherArgs); err != nil {
		action.Release()
		actions.Release()
		return err
	}

	action.Release()
	actions.Release()

	if cfg.IsSystem {
		_, err = oleutil.CallMethod(
			rootFolder,
			"RegisterTaskDefinition",
			cfg.TaskName,
			taskDefinition,
			TaskCreateOrUpdate,
			nil,
			nil,
			TaskLogonNone,
		)
	} else {
		_, err = oleutil.CallMethod(
			rootFolder,
			"RegisterTaskDefinition",
			cfg.TaskName,
			taskDefinition,
			TaskCreateOrUpdate,
			cfg.UserContext,
			nil,
			TaskLogonInteractive,
		)
	}

	if err != nil {
		return err
	}

	taskProg, err := oleutil.CallMethod(rootFolder, "GetTask", cfg.TaskName)
	if err == nil {
		taskObj := taskProg.ToIDispatch()
		oleutil.CallMethod(taskObj, "Run", nil)
		taskObj.Release()
	}

	return nil
}

// =========================================================================
// 5. Context-Aware WMI Verification
// =========================================================================

// checkWMIForProcess queries WMI to determine if AutoHotkey64.exe is running with the specified script path.
func checkWMIForProcess(scriptPath string) bool {
	if err := ole.CoInitialize(0); err != nil {
		return false
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return false
	}
	defer unknown.Release()

	wmiUnknown, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return false
	}
	defer wmiUnknown.Release()

	serviceProg, err := oleutil.CallMethod(wmiUnknown, "ConnectServer", nil, "ROOT\\CIMV2")
	if err != nil {
		return false
	}
	service := serviceProg.ToIDispatch()
	defer service.Release()

	query := "SELECT CommandLine FROM Win32_Process WHERE Name = 'AutoHotkey64.exe'"
	resultProg, err := oleutil.CallMethod(service, "ExecQuery", query)
	if err != nil {
		return false
	}
	results := resultProg.ToIDispatch()
	defer results.Release()

	enum, err := results.GetProperty("_NewEnum")
	if err != nil {
		return false
	}
	defer enum.Clear()

	ienumUnknown, err := enum.ToIUnknown().QueryInterface(ole.IID_IEnumVariant)
	if err != nil {
		return false
	}

	ienum := (*ole.IEnumVARIANT)(ienumUnknown)
	defer ienum.Release()

	for {
		// ienum.Next takes 1 argument and returns 3 values
		variant, fetched, err := ienum.Next(1)
		if err != nil || fetched == 0 {
			break
		}

		process := variant.ToIDispatch()
		cmdLineProg, err := oleutil.GetProperty(process, "CommandLine")
		if err == nil {
			if strings.Contains(cmdLineProg.ToString(), scriptPath) {
				process.Release()
				variant.Clear()
				return true
			}
		}
		process.Release()
		variant.Clear()
	}
	return false
}

// verifyAHKRunning checks every second if AutoHotkey64.exe is running with the specified script path.
// It utilizes Go context for standardized timeout polling
func verifyAHKRunning(ctx context.Context, scriptPath string) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done(): // Timeout reached
			return false
		case <-ticker.C: // Tick every 1 second
			if checkWMIForProcess(scriptPath) {
				return true
			}
		}
	}
}

// terminateSpecificAHKScript uses WMI to surgically kill only the AHK process running our script
func terminateSpecificAHKScript(scriptPath string) (int, error) {
	if err := ole.CoInitialize(0); err != nil {
		return 0, err
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return 0, err
	}
	defer unknown.Release()

	wmiUnknown, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return 0, err
	}
	defer wmiUnknown.Release()

	serviceProg, err := oleutil.CallMethod(wmiUnknown, "ConnectServer", nil, "ROOT\\CIMV2")
	if err != nil {
		return 0, err
	}

	service := serviceProg.ToIDispatch()
	defer service.Release()

	query := "SELECT * FROM Win32_Process WHERE Name = 'AutoHotkey64.exe'"

	resultProg, err := oleutil.CallMethod(service, "ExecQuery", query)
	if err != nil {
		return 0, err
	}

	results := resultProg.ToIDispatch()
	defer results.Release()

	enum, err := results.GetProperty("_NewEnum")
	if err != nil {
		return 0, err
	}
	defer enum.Clear()

	ienumUnknown, err := enum.ToIUnknown().QueryInterface(ole.IID_IEnumVariant)
	if err != nil {
		return 0, err
	}

	ienum := (*ole.IEnumVARIANT)(ienumUnknown)
	defer ienum.Release()

	terminatedCount := 0

	for {
		variant, fetched, err := ienum.Next(1)
		if err != nil || fetched == 0 {
			break
		}

		process := variant.ToIDispatch()

		cmdLineProg, err := oleutil.GetProperty(process, "CommandLine")
		if err == nil {
			if strings.Contains(cmdLineProg.ToString(), scriptPath) {
				if _, err := oleutil.CallMethod(process, "Terminate"); err == nil {
					terminatedCount++
				}
			}
		}

		process.Release()
		variant.Clear()
	}

	return terminatedCount, nil
}

// =========================================================================
// 6. Execution Code
// =========================================================================

// main orchestrates the deployment process, handling configuration, environment setup, asset staging, COM registration, and verification.
func main() {

	// Configuration Check
	cfg, err := buildConfig()
	if err != nil {
		log.Fatalf("Fatal: Failed to build configuration: %v", err)
	}

	// Privilege Check
	if cfg.IsSystem && !isAdmin() {
		log.Fatalf("Elevation Required: Please run this installer as an Administrator or use --mode=user.")
	}

	// -------------------------------------------------------------------------
	// Environment Preparation
	// -------------------------------------------------------------------------
	if err := os.MkdirAll(cfg.TargetDir, 0755); err != nil {
		log.Fatalf("Fatal: Could not create target directory: %v", err)
	}

	// Setup dual-logging (Stdout + File)
	logFile := setupFileLogging(cfg.LogPath)
	if logFile != nil {
		defer logFile.Close()
	}

	log.Printf("Starting Deployment Manager (Mode: %s)", cfg.Mode)
	log.Printf("Target Directory: %s", cfg.TargetDir)
	log.Printf("Installer Log: %s", cfg.LogPath)

	// -------------------------------------------------------------------------
	// AutoHotkey Engine Deployment
	// -------------------------------------------------------------------------

	// Check if AutoHotkey64.exe exists; if not, download and extract it
	if !fileExists(cfg.AHKExe) {
		log.Println("AutoHotkey64.exe not found locally. Downloading from official source...")
		if err := os.MkdirAll(cfg.AHKDir, 0755); err != nil {
			log.Fatalf("Fatal: Could not create AutoHotkey directory: %v", err)
		}

		if err := downloadFile(ahkZipURL, cfg.ZipPath); err != nil {
			log.Fatalf("Deployment Failed: Could not download AutoHotkey engine: %v", err)
		}

		log.Printf("Extracting AutoHotkey to %s...", cfg.AHKDir)
		if err := unzip(cfg.ZipPath, cfg.AHKDir); err != nil {
			log.Fatalf("Deployment Failed: Extraction error: %v", err)
		}
		if err := os.Remove(cfg.ZipPath); err != nil {
			log.Printf("Warning: Could not remove temporary zip file %s: %v", cfg.ZipPath, err)
		}
		log.Println("AutoHotkey engine successfully installed.")
	} else {
		log.Println("AutoHotkey engine is already installed locally.")
	}

	// -------------------------------------------------------------------------
	// Stage Assets
	// -------------------------------------------------------------------------

	// Define local paths for the AHK script and Launcher binary
	localAHKScript := filepath.Join(cfg.ExeDir, "ErgonomicMouse.ahk")
	localLauncher := filepath.Join(cfg.ExeDir, "Launcher.exe")

	// Copy AHK Script
	if fileExists(localAHKScript) {
		log.Printf("Copying AHK script to %s...", cfg.TargetDir)
		if err := copyFile(localAHKScript, cfg.ScriptDest); err != nil {
			log.Fatalf("Fatal: Failed to copy AHK script: %v", err)
		}
	} else {
		log.Fatalf("Warning: Could not find 'ErgonomicMouse.ahk' in the installer folder. Aborting.")
	}

	// Copy Launcher binary
	if fileExists(localLauncher) {
		log.Printf("Copying unified auto-updater engine to %s...", cfg.TargetDir)
		if err := copyFile(localLauncher, cfg.LauncherDest); err != nil {
			log.Fatalf("Fatal: Failed to copy Launcher binary: %v", err)
		}
	} else {
		log.Fatalf("Warning: Could not find 'Launcher.exe' in the installer folder. Aborting.")
	}

	// Terminate Stale AHK Processes
	terminatedCount, err := terminateSpecificAHKScript(cfg.ScriptDest)
	if err != nil {
		log.Printf("Warning: Could not terminate stale AutoHotkey process: %v", err)
	} else if terminatedCount > 0 {
		log.Printf("Terminated %d stale AutoHotkey process(es).", terminatedCount)
	}

	// -------------------------------------------------------------------------
	// Unified Registration
	// -------------------------------------------------------------------------
	log.Printf("Registering %s scheduled task via Windows COM API: '%s'...", cfg.Mode, cfg.TaskName)

	if err := registerTaskWithCOM(cfg); err != nil {
		log.Fatalf("Deployment Failed: COM registration error: %v", err)
	}

	log.Printf("Success! Native COM Task '%s' is registered and triggered.", cfg.TaskName)
	log.Println("Waiting for process startup...")

	// -------------------------------------------------------------------------
	// Context-Aware Verification Loop
	// -------------------------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if verifyAHKRunning(ctx, cfg.ScriptDest) {
		log.Printf("Verification Success: 'AutoHotkey64.exe' is running your script.")
	} else {
		log.Printf("Warning: Verification Timeout: Task was triggered, but AutoHotkey did not initialize within 15 seconds.")
	}

	log.Println("Deployment Completed successfully.")
	time.Sleep(2 * time.Second) // Brief pause so manual runners can read the final terminal output
}
