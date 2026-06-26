package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// =========================================================================
// 1. Declarations (Constants)
// =========================================================================

const ahkZipUrl = "https://www.autohotkey.com/download/ahk-v2.zip"

// =========================================================================
// 2. Variables (Global State)
// =========================================================================

var (
	targetDir           string
	ahkDir              string
	ahkExecutable       string
	ahkScriptTargetPath string
	launcherTargetPath  string
	zipPath             string
	exeDir              string
	taskName            string
	description         string
	userContext         string
)

// =========================================================================
// 3. Helper Functions
// =========================================================================

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

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

func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

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

// -------------------------------------------------------------------------
// COM Task Registration (System Mode)
// -------------------------------------------------------------------------
func registerSystemTaskWithCOM(name, targetPath, args, desc string) error {
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

	oleutil.CallMethod(rootFolder, "DeleteTask", name, 0)

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
	oleutil.PutProperty(regInfo, "Description", desc)
	regInfo.Release()

	principalProg, err := oleutil.GetProperty(taskDefinition, "Principal")
	if err != nil {
		return err
	}
	principal := principalProg.ToIDispatch()
	oleutil.PutProperty(principal, "GroupId", "Builtin\\Users")
	oleutil.PutProperty(principal, "RunLevel", 1) // Elevated
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
	oleutil.CallMethod(triggers, "Create", 9)
	triggers.Release()

	actionsProg, err := oleutil.GetProperty(taskDefinition, "Actions")
	if err != nil {
		return err
	}
	actions := actionsProg.ToIDispatch()
	actionNewProg, err := oleutil.CallMethod(actions, "Create", 0)
	if err != nil {
		actions.Release()
		return err
	}
	action := actionNewProg.ToIDispatch()
	oleutil.PutProperty(action, "Path", targetPath)
	oleutil.PutProperty(action, "Arguments", args) // Injects the --mode flag!
	action.Release()
	actions.Release()

	_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", name, taskDefinition, 6, nil, nil, 0)
	if err != nil {
		return err
	}

	taskProg, err := oleutil.CallMethod(rootFolder, "GetTask", name)
	if err == nil {
		taskObj := taskProg.ToIDispatch()
		oleutil.CallMethod(taskObj, "Run", nil)
		taskObj.Release()
	}

	return nil
}

// -------------------------------------------------------------------------
// COM Task Registration (User Mode)
// -------------------------------------------------------------------------
func registerUserTaskWithCOM(name, targetPath, args, desc, user string) error {
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

	oleutil.CallMethod(rootFolder, "DeleteTask", name, 0)

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
	oleutil.PutProperty(regInfo, "Description", desc)
	regInfo.Release()

	principalProg, err := oleutil.GetProperty(taskDefinition, "Principal")
	if err != nil {
		return err
	}
	principal := principalProg.ToIDispatch()
	oleutil.PutProperty(principal, "UserId", user)
	oleutil.PutProperty(principal, "LogonType", 3)
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
	triggerProg, err := oleutil.CallMethod(triggers, "Create", 9)
	if err != nil {
		triggers.Release()
		return err
	}
	trigger := triggerProg.ToIDispatch()
	oleutil.PutProperty(trigger, "UserId", user)
	trigger.Release()
	triggers.Release()

	actionsProg, err := oleutil.GetProperty(taskDefinition, "Actions")
	if err != nil {
		return err
	}
	actions := actionsProg.ToIDispatch()
	actionNewProg, err := oleutil.CallMethod(actions, "Create", 0)
	if err != nil {
		actions.Release()
		return err
	}
	action := actionNewProg.ToIDispatch()
	oleutil.PutProperty(action, "Path", targetPath)
	oleutil.PutProperty(action, "Arguments", args) // Injects the --mode flag!
	action.Release()
	actions.Release()

	_, err = oleutil.CallMethod(rootFolder, "RegisterTaskDefinition", name, taskDefinition, 6, user, nil, 3)
	if err != nil {
		return err
	}

	taskProg, err := oleutil.CallMethod(rootFolder, "GetTask", name)
	if err == nil {
		taskObj := taskProg.ToIDispatch()
		oleutil.CallMethod(taskObj, "Run", nil)
		taskObj.Release()
	}

	return nil
}

// -----------------------------------------------------------------------------------------------------------
// WMI task to verify if the AHK script is running with the exact command line of running AutoHotkey processes
// -----------------------------------------------------------------------------------------------------------

func verifyAHKRunning(scriptPath string) bool {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return false
	}
	defer unknown.Release()

	// Handle the 2 return values from QueryInterface
	wmiUnknown, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return false
	}
	wmi := wmiUnknown
	defer wmi.Release()

	serviceProg, err := oleutil.CallMethod(wmi, "ConnectServer", nil, "ROOT\\CIMV2")
	if err != nil {
		return false
	}
	service := serviceProg.ToIDispatch()
	defer service.Release()

	// Query for running AutoHotkey processes
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

	// Use correct case "IID_IEnumVariant"
	ienumUnknown, err := enum.ToIUnknown().QueryInterface(ole.IID_IEnumVariant)
	if err != nil {
		return false
	}

	// Explicitly cast to EnumVariant so we can use .Next()
	ienum := (*ole.IEnumVARIANT)(ienumUnknown)
	defer ienum.Release()

	for {
		variant, length, err := ienum.Next(1)
		if err != nil || length == 0 {
			break
		}

		process := variant.ToIDispatch()
		if process == nil {
			variant.Clear()
			continue
		}

		cmdLineProg, err := oleutil.GetProperty(process, "CommandLine")
		if err == nil {
			if strings.Contains(cmdLineProg.ToString(), scriptPath) {
				cmdLineProg.Clear()
				process.Release()
				variant.Clear()
				return true
			}
			cmdLineProg.Clear()
		}

		process.Release()
		variant.Clear()
	}

	return false
}

// =========================================================================
// 4. Execution Code
// =========================================================================

func main() {
	// -------------------------------------------------------------------------
	// CLI Flag Parsing & Path Configuration
	// -------------------------------------------------------------------------
	mode := flag.String("mode", "system", "Installation mode: 'system' or 'user'")
	flag.Parse()

	executablePath, err := os.Executable()
	if err == nil {
		exeDir = filepath.Dir(executablePath)
	} else {
		exeDir = "."
	}

	if *mode == "user" {
		targetDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "ErgonomicMouse")
		ahkDir = filepath.Join(targetDir, "AutoHotkey")
		taskName = "ErgonomicMouseMapping-User"
		description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for the current user."

		userDomain := os.Getenv("USERDOMAIN")
		userName := os.Getenv("USERNAME")
		if userDomain != "" && userName != "" {
			userContext = userDomain + "\\" + userName
		} else {
			userContext = userName
		}
	} else {
		if !isAdmin() {
			fmt.Println("Elevation Required: Please run this installer as an Administrator or use --mode=user.")
			time.Sleep(3 * time.Second)
			return
		}
		targetDir = filepath.Join(os.Getenv("ProgramData"), "ErgonomicMouse")
		ahkDir = filepath.Join(os.Getenv("ProgramFiles"), "AutoHotkey", "v2")
		taskName = "ErgonomicMouseMapping"
		description = "Runs the ergonomic mouse remapping script (F5/F6/F7) for all users."
	}

	ahkExecutable = filepath.Join(ahkDir, "AutoHotkey64.exe")
	ahkScriptTargetPath = filepath.Join(targetDir, "ErgonomicMouse.ahk")
	launcherTargetPath = filepath.Join(targetDir, "Launcher.exe")
	zipPath = filepath.Join(targetDir, "ahk-v2.zip")
	launcherArgs := fmt.Sprintf("--mode=%s", *mode)

	// -------------------------------------------------------------------------
	// Environment Preparation
	// -------------------------------------------------------------------------
	fmt.Printf("Creating target directory at %s...\n", targetDir)
	os.MkdirAll(targetDir, 0755)

	if !fileExists(ahkExecutable) {
		fmt.Println("Downloading AutoHotkey engine from official source...")
		os.MkdirAll(ahkDir, 0755)

		err := downloadFile(ahkZipUrl, zipPath)
		if err != nil {
			fmt.Printf("Deployment Failed: Could not download AutoHotkey engine: %v\n", err)
			time.Sleep(3 * time.Second)
			return
		}

		fmt.Printf("Extracting AutoHotkey to %s...\n", ahkDir)
		err = unzip(zipPath, ahkDir)
		os.Remove(zipPath)

		if err != nil {
			fmt.Printf("Deployment Failed: Extraction error: %v\n", err)
			time.Sleep(3 * time.Second)
			return
		}
		fmt.Println("AutoHotkey engine successfully installed.")
	} else {
		fmt.Println("AutoHotkey engine is already installed locally.")
	}

	// -------------------------------------------------------------------------
	// Stage Assets
	// -------------------------------------------------------------------------
	localAHKScript := filepath.Join(exeDir, "ErgonomicMouse.ahk")
	localLauncher := filepath.Join(exeDir, "Launcher.exe")

	if fileExists(localAHKScript) {
		fmt.Printf("Copying AHK script to %s...\n", targetDir)
		copyFile(localAHKScript, ahkScriptTargetPath)
	} else {
		fmt.Println("Warning: Could not find 'ErgonomicMouse.ahk' in the installer folder. Aborting.")
		time.Sleep(3 * time.Second)
		return
	}

	if fileExists(localLauncher) {
		fmt.Printf("Copying unified auto-updater engine to %s...\n", targetDir)
		copyFile(localLauncher, launcherTargetPath)
	} else {
		fmt.Println("Warning: Could not find 'Launcher.exe' in the installer folder. Aborting.")
		time.Sleep(3 * time.Second)
		return
	}

	exec.Command("taskkill", "/F", "/IM", "AutoHotkey64.exe").Run()

	// -------------------------------------------------------------------------
	// Registration Routing
	// -------------------------------------------------------------------------
	fmt.Printf("Registering %s scheduled task via Windows COM API: '%s'...\n", *mode, taskName)

	if *mode == "user" {
		err = registerUserTaskWithCOM(taskName, launcherTargetPath, launcherArgs, description, userContext)
	} else {
		err = registerSystemTaskWithCOM(taskName, launcherTargetPath, launcherArgs, description)
	}

	if err != nil {
		fmt.Printf("Deployment Failed: COM registration error: %v\n", err)
		time.Sleep(5 * time.Second)
		return
	}

	fmt.Printf("Success! Native COM Task '%s' is registered and triggered.\n", taskName)
	fmt.Println("Waiting for process startup...")

	// Native Verification Loop
	found := false
	for i := 0; i < 15; i++ {
		if verifyAHKRunning(ahkScriptTargetPath) {
			fmt.Printf("Verification Success: 'AutoHotkey64.exe' is running your script.\n")
			found = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !found {
		fmt.Println("Warning: Verification Timeout: Task was triggered, but AutoHotkey did not initialize within 15 seconds.")
	}

	fmt.Println("Deployment Completed successfully.")
	time.Sleep(2 * time.Second)
}
