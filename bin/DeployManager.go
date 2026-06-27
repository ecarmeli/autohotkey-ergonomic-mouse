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
	ahkZipUrl = "https://www.autohotkey.com/download/ahk-v2.zip"

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
		cfg.IsSystem = false
		cfg.TargetDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "ErgonomicMouse")
		cfg.AHKDir = filepath.Join(cfg.TargetDir, "AutoHotkey")
		cfg.TaskName = "ErgonomicMouseMapping-User"
		cfg.Description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for the current user."

		domain := os.Getenv("USERDOMAIN")
		user := os.Getenv("USERNAME")
		if domain != "" && user != "" {
			cfg.UserContext = domain + "\\" + user
		} else {
			cfg.UserContext = user
		}
	} else {
		cfg.IsSystem = true
		cfg.TargetDir = filepath.Join(os.Getenv("ProgramData"), "ErgonomicMouse")
		cfg.AHKDir = filepath.Join(os.Getenv("ProgramFiles"), "AutoHotkey", "v2")
		cfg.TaskName = "ErgonomicMouseMapping"
		cfg.Description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for all users."
	}

	cfg.AHKExe = filepath.Join(cfg.AHKDir, "AutoHotkey64.exe")
	cfg.ScriptDest = filepath.Join(cfg.TargetDir, "ErgonomicMouse.ahk")
	cfg.LauncherDest = filepath.Join(cfg.TargetDir, "Launcher.exe")
	cfg.ZipPath = filepath.Join(cfg.TargetDir, "ahk-v2.zip")

	return cfg, nil
}

// =========================================================================
// 3. Helper Functions
// =========================================================================

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
func setupFileLogging(targetDir string) *os.File {
	logPath := filepath.Join(targetDir, "deploy_manager.log")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		log.SetOutput(os.Stdout)
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

// downloadFile downloads a file from the given URL to the specified final path.
func downloadFile(url string, finalPath string) error {
	// Use a custom client with a 60-second timeout to prevent hangs
	client := &http.Client{Timeout: 60 * time.Second}

	// Download to a temporary file first
	tmpPath := finalPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	resp, err := client.Get(url)
	if err != nil {
		out.Close()
		os.Remove(tmpPath) // Clean up partial file
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close() // Must explicitly close before renaming

	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename guarantees the system only sees a 100% complete file
	return os.Rename(tmpPath, finalPath)
}

// unzip extracts a zip archive to the specified destination directory, ensuring no path traversal occurs.
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure our destination path is absolute and clean
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(destAbs, f.Name)

		// ZIP SLIP HARDENING: Ensure the extracted path stays strictly inside the target directory
		if !strings.HasPrefix(fpath, filepath.Clean(destAbs)+string(os.PathSeparator)) {
			return fmt.Errorf("Illegal file path in zip: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
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

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// =========================================================================
// 4. Unified COM Task Registration (DRY Principle)
// =========================================================================

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

	_, err = oleutil.CallMethod(service, "Connect")
	if err != nil {
		return err
	}

	rootFolderProg, err := oleutil.CallMethod(service, "GetFolder", "\\")
	if err != nil {
		return err
	}
	rootFolder := rootFolderProg.ToIDispatch()
	defer rootFolder.Release()

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
	oleutil.PutProperty(regInfo, "Description", cfg.Description)
	regInfo.Release()

	principalProg, err := oleutil.GetProperty(taskDefinition, "Principal")
	if err != nil {
		return err
	}
	principal := principalProg.ToIDispatch()

	// Mode-specific Principal Routing
	if cfg.IsSystem {
		oleutil.PutProperty(principal, "GroupId", "Builtin\\Users")
		oleutil.PutProperty(principal, "RunLevel", TaskRunLevelHighest)
	} else {
		oleutil.PutProperty(principal, "UserId", cfg.UserContext)
		oleutil.PutProperty(principal, "LogonType", TaskLogonInteractive)
	}
	principal.Release()

	settingsProg, err := oleutil.GetProperty(taskDefinition, "Settings")
	if err != nil {
		return err
	}
	settings := settingsProg.ToIDispatch()
	oleutil.PutProperty(settings, "AllowStartIfOnBatteries", true)
	oleutil.PutProperty(settings, "StopIfGoingOnBatteries", false)
	oleutil.PutProperty(settings, "ExecutionTimeLimit", "PT0S")
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
	// Only bind to specific user if in user-mode
	if !cfg.IsSystem {
		oleutil.PutProperty(trigger, "UserId", cfg.UserContext)
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
	oleutil.PutProperty(action, "Path", cfg.LauncherDest)
	oleutil.PutProperty(action, "Arguments", cfg.LauncherArgs)
	action.Release()
	actions.Release()

	// Mode-specific Registration Routing
	if cfg.IsSystem {
		_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", cfg.TaskName, taskDefinition, TaskCreateOrUpdate, nil, nil, TaskLogonNone)
	} else {
		_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", cfg.TaskName, taskDefinition, TaskCreateOrUpdate, cfg.UserContext, nil, TaskLogonInteractive)
	}
	if err != nil {
		return err
	}

	// Trigger execution immediately
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
	ole.CoInitialize(0)
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
	cfg, err := buildConfig()
	if err != nil {
		log.Fatalf("Fatal: Failed to build configuration: %v", err)
	}

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
	logFile := setupFileLogging(cfg.TargetDir)
	if logFile != nil {
		defer logFile.Close()
	}

	log.Printf("Starting Deployment Manager (Mode: %s)", cfg.Mode)
	log.Printf("Target Directory: %s", cfg.TargetDir)

	if !fileExists(cfg.AHKExe) {
		log.Println("AutoHotkey64.exe not found locally. Downloading from official source...")
		if err := os.MkdirAll(cfg.AHKDir, 0755); err != nil {
			log.Fatalf("Fatal: Could not create AutoHotkey directory: %v", err)
		}

		if err := downloadFile(ahkZipUrl, cfg.ZipPath); err != nil {
			log.Fatalf("Deployment Failed: Could not download AutoHotkey engine: %v", err)
		}

		log.Printf("Extracting AutoHotkey to %s...", cfg.AHKDir)
		if err := unzip(cfg.ZipPath, cfg.AHKDir); err != nil {
			log.Fatalf("Deployment Failed: Extraction error: %v", err)
		}
		os.Remove(cfg.ZipPath)
		log.Println("AutoHotkey engine successfully installed.")
	} else {
		log.Println("AutoHotkey engine is already installed locally.")
	}

	// -------------------------------------------------------------------------
	// Stage Assets
	// -------------------------------------------------------------------------
	localAHKScript := filepath.Join(cfg.ExeDir, "ErgonomicMouse.ahk")
	localLauncher := filepath.Join(cfg.ExeDir, "Launcher.exe")

	if fileExists(localAHKScript) {
		log.Printf("Copying AHK script to %s...", cfg.TargetDir)
		if err := copyFile(localAHKScript, cfg.ScriptDest); err != nil {
			log.Fatalf("Fatal: Failed to copy AHK script: %v", err)
		}
	} else {
		log.Fatalf("Warning: Could not find 'ErgonomicMouse.ahk' in the installer folder. Aborting.")
	}

	if fileExists(localLauncher) {
		log.Printf("Copying unified auto-updater engine to %s...", cfg.TargetDir)
		if err := copyFile(localLauncher, cfg.LauncherDest); err != nil {
			log.Fatalf("Fatal: Failed to copy Launcher binary: %v", err)
		}
	} else {
		log.Fatalf("Warning: Could not find 'Launcher.exe' in the installer folder. Aborting.")
	}

	// Terminate specific stale sessions safely (Fixing the variable name here!)
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
