# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping featuring automated background update detection.

This project packages everything into a unified Windows installer executable (`ErgonomicMouseSetup.exe`) that handles clean deployments, native privilege handoffs, and silent uninstallation directly from the Windows App settings.

## 📥 Quick Start
1. Download the latest `ErgonomicMouseSetup.exe` from the repository [Releases](https://github.com/ecarmeli/autohotkey-ergonomic-mouse/releases).
2. Double-click the installer to launch the setup wizard.
   * *Note on SmartScreen:* Because this is a free, open-source project, the installer is not signed with a commercial Authenticode certificate. If Windows SmartScreen appears, click **More info** followed by **Run anyway** to proceed.
3. Follow the wizard instructions.

---

## 🩺 Why This Project Exists: Vision & Objectives

* **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger/wrist strain and Repetitive Strain Injuries (RSI).
* **The Solution:** Shifting primary mouse clicks directly to the keyboard's middle Function keys (`F5`-`F7`). By separating cursor movement from clicking, this project creates a balanced, two-handed workflow that minimizes mechanical fatigue on your primary joints.

---

## ✨ Key Features & Ergonomic Design

* **RSI Strain Relief:** Balances physical workload across both hands by moving high-frequency clicking tasks away from the mouse.
* **Tendon Protection:** Holding down a key triggers a single continuous mouse click instead of sending rapid, exhausting inputs.
* **Frictionless Drag-and-Drop:** An automatic 2px micro-movement triggers on initial click-hold, forcing picky applications or IDEs to register drag actions instantly without requiring a tense, heavy grip.
* **Smooth Side-Scrolling:** Use `Shift + Scroll Wheel` to pan horizontally across wide data structures, codebases, or spreadsheets seamlessly.
* **Instant Master Toggle:** Uses the physical `Scroll Lock` key (and its native hardware LED) as a global toggle to seamlessly transition between mouse mode and standard typing.

---

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

---

## 💻 Compatibility & System Requirements

This solution is engineered to play nicely out of the box with core enterprise productivity software and development tools (e.g., Microsoft Edge, Office Suite, Visual Studio Code).

* **Operating System:** Microsoft Windows 10 or Windows 11 (64-bit architecture required).
* **Dependencies:** None. The installer bundles core binary and compiled Go launcher modules out of the box.
* **Hardware Interactivity:** Mappings take advantage of standard peripheral inputs. The Master Toggle relies on a physical `Scroll Lock` key layout; systems missing this physical key can trigger it via standard virtual keyboard overlays or alternate custom remappings.

---

## 🛠️ Installation & Privileges

The installation wizard dynamically adapts its execution layer based on user privilege:

* **Current User (Standard Privilege):** Installs strictly within `%LocalAppData%\ErgonomicMouse` and registers interactive logon tasks. Ideal for restricted corporate environments without administrative rights.
* **All Users (Elevated Privilege):** Installs globally to `%ProgramData%\ErgonomicMouse` and registers a Task Scheduler logon task intended to support interaction with elevated applications. 

---

## 🔐 Security Model

The project enforces strict state management and separates runtime executables from user-writable operational data to prevent privilege escalation vectors.

* **Cross-Scope Protection:** The installer enforces atomic execution contexts. To prevent orphaned deployments, it explicitly blocks installation if launched manually as an Administrator but configured for a "Current User" deployment. It also actively scans for existing user-mode installations across profiles to prevent split-brain duplications when elevating.
* **Hermetic Directories:** In System Mode, runtime files are installed under `%ProgramData%\ErgonomicMouse`. This directory is actively hardened so standard users receive read/execute access only, preventing unauthorized modification of program files.
* **Safe Log Routing:** Operational logs (for both the execution engine and the deployment manager) are never written to the protected system installation directory. They are securely routed to the interactive user's local profile at:
  %LocalAppData%\ErgonomicMouse\logs\launcher.log
  %LocalAppData%\ErgonomicMouse\logs\deploymanager.log

---

## 🔄 Update & Lifecycle Model

Ergonomic Mouse does not silently download or replace runtime files during startup.

`Launcher.exe` performs passive update detection only: it checks the latest published GitHub release, compares it with the installed version, and logs when a newer installer is available. Updates are delivered strictly through the full `ErgonomicMouseSetup.exe` installer so all components stay version-aligned.

The installer fully supports clean uninstallation and lifecycle management directly from the standard Windows Apps settings menu.

---

## 📂 Repository Structure

```text
autohotkey-ergonomic-mouse/
│
├── .github/
│   ├── workflows/
│   │   ├── security-and-quality.yml   # CI: compilation validation, linting, security scanning
│   │   └── build-and-release.yml      # CD: Automated Go compilation & Inno Setup packaging
│   └── dependabot.yml
│
├── cmd/
│   ├── launcher/
│   │   └── main.go                    # Entry point binary; launches AHK and performs update detection
│   └── deploymanager/ 
│       └── main.go                    # Deployment engine; configures COM tasks & system ACLs
│ 
├── src/
│   └── ErgonomicMouse.ahk             # Runtime AutoHotkey source script
│
├── installer.iss                      # Inno Setup blueprint compiler configuration
├── go.mod                             # Go module definition
├── go.sum                             # Dependency lock file
├── .gitignore
└── README.md
```
---

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

---

## 🛡️ CI/CD Pipeline & Security Gating

The project features an automated, multi-tier GitHub Actions delivery structure utilizing cryptographically pinned action dependencies:

### 1. Security & Quality Pipeline (`security-and-quality.yml`)
* Runs on every pull request to protect code integrity.
* **Checks Include:** `go mod verify` module integrity checking, `go vet` static analysis, `staticcheck` advanced linter execution, `govulncheck` code vulnerability dependency tracking, and `trivy` scanning for secrets and repository configurations.

### 2. Build & Release Pipeline (`build-and-release.yml`)
* Orchestrates an automated, 3-stage delivery pipeline split across isolated platforms for optimized execution:
  * **Stage 1: Build & Package (Windows):** Runs within a read-only permissions scope. It verifies the stable AutoHotkey core distribution via SHA256 hashes, injects runtime metadata into the Go binaries, compiles the executables, and packages them via Inno Setup. The unverified binaries and installer are saved to secure workflow storage.
  * **Stage 2: Independent Malware Scan (Linux):** Downloads the raw binaries and compiled installer into an isolated container environment. It executes targeted signature scans using **ClamAV** and **YARA**, followed by a dynamic behavioral capability analysis using Mandiant's **Capa** engine. If any vulnerabilities or suspicious capabilities are discovered, the step returns a fatal error, forcing a "fail-closed" termination.
  * **Stage 3: Conditional Production Release (Linux):** If and only if the malware and behavioral scans pass cleanly, this final stage pulls down the verified artifacts, auto-generates release notes, and publishes a formal, public production release asset.