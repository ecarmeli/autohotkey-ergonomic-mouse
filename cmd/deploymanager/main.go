//go:build windows

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "none"
)

const (
	TaskActionExec       = 0
	TaskTriggerLogon     = 9
	TaskCreateOrUpdate   = 6
	TaskLogonInteractive = 3
	TaskLogonNone        = 0
	TaskRunLevelHighest  = 1
)

type Config struct {
	Mode         string
	TargetDir    string
	AHKDir       string
	AHKExe       string
	ScriptDest   string
	LauncherDest string
	ExeDir       string
	TaskName     string
	Description  string
	UserContext  string
	IsSystem     bool
	LauncherArgs string
}

func buildConfig(mode string) (*Config, error) {
	if mode != "system" && mode != "user" {
		return nil, fmt.Errorf("invalid mode %q: expected 'system' or 'user'", mode)
	}

	exePath, err := os.Executable()
	exeDir := "."
	if err == nil {
		exeDir = filepath.Dir(exePath)
	}

	cfg := &Config{
		Mode:         mode,
		ExeDir:       exeDir,
		LauncherArgs: fmt.Sprintf("--mode=%s", mode),
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

		cfg.IsSystem = true
		cfg.TargetDir = filepath.Join(programData, "ErgonomicMouse")
		cfg.AHKDir = filepath.Join(cfg.TargetDir, "AutoHotkey")
		cfg.TaskName = "ErgonomicMouseMapping"
		cfg.Description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for all users."
	}

	cfg.AHKExe = filepath.Join(cfg.AHKDir, "AutoHotkey64.exe")
	cfg.ScriptDest = filepath.Join(cfg.TargetDir, "ErgonomicMouse.ahk")
	cfg.LauncherDest = filepath.Join(cfg.TargetDir, "Launcher.exe")

	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

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

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

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
	putCOMProperty(regInfo, "Description", cfg.Description)
	regInfo.Release()

	principalProg, err := oleutil.GetProperty(taskDefinition, "Principal")
	if err != nil {
		return err
	}
	principal := principalProg.ToIDispatch()

	if cfg.IsSystem {
		putCOMProperty(principal, "GroupId", "Builtin\\Users")
		putCOMProperty(principal, "RunLevel", TaskRunLevelHighest)
	} else {
		putCOMProperty(principal, "UserId", cfg.UserContext)
		putCOMProperty(principal, "LogonType", TaskLogonInteractive)
	}
	principal.Release()

	settingsProg, err := oleutil.GetProperty(taskDefinition, "Settings")
	if err != nil {
		return err
	}
	settings := settingsProg.ToIDispatch()
	putCOMProperty(settings, "DisallowStartIfOnBatteries", false)
	putCOMProperty(settings, "StopIfGoingOnBatteries", false)
	putCOMProperty(settings, "ExecutionTimeLimit", "PT0S")
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
		putCOMProperty(trigger, "UserId", cfg.UserContext)
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

	putCOMProperty(action, "Path", cfg.LauncherDest)
	putCOMProperty(action, "Arguments", cfg.LauncherArgs)
	action.Release()
	actions.Release()

	if cfg.IsSystem {
		_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", cfg.TaskName, taskDefinition, TaskCreateOrUpdate, nil, nil, TaskLogonNone)
	} else {
		_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", cfg.TaskName, taskDefinition, TaskCreateOrUpdate, cfg.UserContext, nil, TaskLogonInteractive)
	}

	if err != nil {
		return err
	}

	taskProg, err := oleutil.CallMethod(rootFolder, "GetTask", cfg.TaskName)
	if err == nil {
		taskObj := taskProg.ToIDispatch()
		taskObj.Release()
	}

	return nil
}

func deleteTaskWithCOM(cfg *Config) error {
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

	_, err = oleutil.CallMethod(rootFolder, "DeleteTask", cfg.TaskName, 0)
	return err
}

func putCOMProperty(dispatch *ole.IDispatch, name string, value interface{}) error {
	if _, err := oleutil.PutProperty(dispatch, name, value); err != nil {
		return fmt.Errorf("failed to set COM property %s: %w", name, err)
	}
	return nil
}

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

func verifyAHKRunning(ctx context.Context, scriptPath string) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if checkWMIForProcess(scriptPath) {
				return true
			}
		}
	}
}

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

func main() {
	mode := flag.String("mode", "system", "Execution mode: 'system' or 'user'")
	installAction := flag.Bool("install", false, "Trigger post-install task registrations")
	uninstallAction := flag.Bool("uninstall", false, "Trigger pre-uninstall system cleanup")
	flag.Parse()

	cfg, err := buildConfig(*mode)
	if err != nil {
		log.Fatalf("Fatal: Failed to build configuration: %v", err)
	}

	// Dynamic Privilege Enforcement
	if cfg.IsSystem && *installAction && !isAdmin() {
		log.Fatalf("Elevation Required: Please relaunch installer inside Administrative context or select User Mode.")
	}

	log.Printf("DeployManager v%s | Mode: %s | Action: install=%t, uninstall=%t", version, cfg.Mode, *installAction, *uninstallAction)

	// --- ACTION: PRE-UNINSTALL CLEANUP ---
	if *uninstallAction {
		log.Println("Action triggered: Cleaning system configurations...")
		terminatedCount, err := terminateSpecificAHKScript(cfg.ScriptDest)
		if err == nil && terminatedCount > 0 {
			log.Printf("Surgically terminated %d active engine runtime process(es).", terminatedCount)
		}
		if err := deleteTaskWithCOM(cfg); err != nil {
			log.Printf("Notice: Task Schedule unregistration bypass: %v", err)
		}
		log.Println("Cleanup tasks finalized successfully.")
		return
	}

	// --- ACTION: POST-INSTALL SYSTEM REGISTRATION ---
	if *installAction {
		log.Println("Action triggered: Provisioning execution endpoints...")

		// Terminate stale background layers safely
		_, _ = terminateSpecificAHKScript(cfg.ScriptDest)

		log.Printf("Registering native COM task scheduler endpoint: %s", cfg.TaskName)
		if err := registerTaskWithCOM(cfg); err != nil {
			log.Fatalf("Deployment Engine Panic: COM registry allocation fault: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if verifyAHKRunning(ctx, cfg.ScriptDest) {
			log.Printf("Verification Success: Remapping container actively listening.")
		} else {
			log.Printf("Warning: Application initiated, context verification timed out.")
		}
		return
	}

	// Safe Fallback for structural checks
	log.Println("Notice: Executed without action parameters. Context validations parsed cleanly.")
}
