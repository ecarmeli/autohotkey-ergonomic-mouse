# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping, featuring seamless, automated background updates.

this project packages everything into a unified Windows installer executable (`ErgonomicMouseSetup.exe`) that handles clean deployments, native privilege handoffs, and silent uninstallation directly from the Windows App settings.

1. Download the latest `ErgonomicMouseSetup.exe` from the repository releases.
2. Double-click the installer and follow the instructions.

## 🩺 Why This Project Exists: Vision & Objectives

> **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger and wrist strain.  
> **The Solution:** By separating cursor movement from clicking, this project creates a balanced, two-handed layout built for long-term joint health and a pain-free workflow.

This project was born out of a vital need for workspace ergonomics and physical strain relief. In a traditional computing setup, the primary mouse hand bears the brunt of continuous, repetitive stress from constant finger clicking — a leading cause of hand fatigue and repetitive strain injuries (RSI).

This solution rebalances that physical workload by shifting primary mouse click actions directly to the keyboard's middle Function keys. By utilizing your secondary hand to execute clicks, you instantly offload the mechanical stress from your mouse hand. This creates a highly efficient, two-handed workflow that minimizes finger strain and promotes a much healthier, more comfortable computing experience.

## 🛠️ Installation & Management

The installation wizard dynamically adapts its user interface based on whether it is running standard or elevated:

* **User Mode (Standard Privilege):**
  * Installs strictly within the current user profile (`%LocalAppData%\ErgonomicMouse`).
  * Registers interactive logon tasks utilizing standard user context tokens.
  * Recommended for restricted corporate environments without administrative privileges.
* **System Mode (Requires Administrator Permission):**
  * Installs globally to machine directories (`%ProgramData%\ErgonomicMouse`).
  * Leverages a native Windows UAC self-elevation handoff block.
  * Installs a high-integrity Task Scheduler logon hook to ensure script functionality remains active even when interacting with administrative applications (e.g., Task Manager, elevated IDEs/terminals).

### Smart Guardrails & Self-Healing Maintenance
* **State Detection:** Double-clicking the installer when an existing installation is detected automatically triggers a silent, structured teardown of background layers before applying updates.
* **Process Sledgehammer:** Uninstallation automatically terminates running loops (`AutoHotkey64.exe` and `Launcher.exe`) to prevent file-lock conflicts, ensuring a 100% clean directory wipe.
* **Centralized Telemetry:** Native installation and deployment logs are captured and automatically mirrored straight to the operational data directory (`\logs\install.log`) for quick diagnostic analysis.

## 🚀 Key Features & Ergonomic Design

* **RSI Strain Relief:** Shifts repetitive click stress from your mouse hand to the keyboard's Function row, instantly balancing the physical workload across both hands to prevent finger and wrist fatigue.
* **Tendon Protection:** Holding down a hotkey triggers just one continuous mouse click instead of sending rapid clicks.
* **Frictionless Drag-and-Drop:** An automatic 2px micro-movement triggers on initial click-hold, forcing picky applications or IDEs to register drag actions instantly without requiring a tense, heavy grip.
* **Horizontal Scrolling:** Bypasses rigid OS scroll constraints to deliver ultra-smooth side-scrolling via `Shift + Scroll Wheel` for effortless timeline or spreadsheet panning.
* **Instant Master Toggle:** Uses the physical `Scroll Lock` key (and its native LED) as a global master switch to seamlessly transition between ergonomic mouse mode and standard typing.
* **"Set and Forget" Automation:** A silent background Go engine uses native COM interfaces to establish persistent logon triggers, managing updates seamlessly with zero user friction.

## 🎮 Keymaps & Controls

| Input | Action | Behavior |
| :--- | :--- | :--- |
| **`Scroll Lock`** | Master Toggle | Enables (LED ON) or Disables (LED OFF) all mappings. |
| **`F5`** | Left Mouse Click | Supports click-and-drag holding + micro-nudging. |
| **`F6`** | Middle Mouse Click | Holds down middle click for canvas panning / hand-scrolling. |
| **`F7`** | Right Mouse Click | Supports click-and-drag holding + micro-nudging. |
| **`Shift + WheelUp`** | Horizontal Scroll Left | High-precision messaging bypasses OS inertia limits. |
| **`Shift + WheelDown`** | Horizontal Scroll Right| High-precision messaging bypasses OS inertia limits. |
| **`Ctrl + F12`** | Panic Release | Instantly forces a release of all virtual mouse buttons if stuck. |

## 📂 Repository Structure

```text
autohotkey-ergonomic-mouse/
│
├── .github/
│   └── workflows/
│       ├── security-and-quality.yml   # CI: compilation validation, linting, security scanning
│       └── build-and-release.yml      # CD: Automated Go compilation & Inno Setup packaging
│
├── bin/                               # Local/CI compilation target directory
│
├── cmd/
│   ├── launcher/
│   │   └── main.go                    # Entry point binary; manages lifecycle/updates
│   └── deploymanager/ 
│       └── main.go                    # State manager engine; configures COM tasks & logs
│ 
├── installer.iss                      # Inno Setup blueprint compiler configuration
├── go.mod                             # Go module definition
├── go.sum                             # Dependency lock file
├── .gitignore
└── README.md
```

## ⚙️ Build & Development

### Local Go Compilation

To compile optimized, production-ready binaries locally without console windows popping into view, pass the optimized GUI link flags:

```powershell
# Declare versioning metadata
$version = "1.0.0"
$buildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
$commit = git rev-parse --short HEAD
$ldflags = "-s -w -H=windowsgui -X main.version=$version -X main.buildTime=$buildTime -X main.gitCommit=$commit"

# Build components
go build -ldflags "$ldflags" -o bin/Launcher.exe ./cmd/launcher
go build -ldflags "$ldflags" -o bin/DeployManager.exe ./cmd/deploymanager
```

### Local Installer Compilation

Ensure you have Inno Setup 6+ installed locally, pull the AutoHotkey binaries into your `.\bin\AutoHotkey` folder, and execute the compiler:

```powershell
& "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" .\installer.iss
```

## 🔄 CI/CD Pipeline

The project features an automated, multi-tier GitHub Actions delivery structure:

### 1. Security & Quality Pipeline (`security-and-quality.yml`)
* Run on every pull request to protect code integrity.
* **Checks Include:** `go vet` static analysis, `staticcheck` advanced linter execution, `govulncheck` code vulnerability dependency tracking, and `trivy` scanning for secrets and repository configurations.

### 2. Build & Release Pipeline (`build-and-release.yml`)
* Automatically spins up a virtual `windows-latest` runner on pushes to the main branch.
* Dynamically downloads and stages stable AutoHotkey core dependency distributions.
* Injects variables (`version`, `buildTime`, `gitCommit`) directly into the compiled Go assets.
* Invokes `ISCC.exe` out-of-the-box to package and export a compiled `ErgonomicMouseSetup.exe` artifact.

## 🔄 Update Model (High-Level)

* Launcher binary is responsible for execution flow
* AutoHotkey script acts as runtime payload
* Updates are fetched and validated before execution
* Designed for:
  * resilience
  * silent operation
  * zero user friction