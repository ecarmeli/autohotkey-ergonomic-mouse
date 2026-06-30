# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping, featuring seamless, automated background updates.

This project packages everything into a unified Windows installer executable (`ErgonomicMouseSetup.exe`) that handles clean deployments, native privilege handoffs, and silent uninstallation directly from the Windows App settings.

## 📥 Quick Start
1. Download the latest `ErgonomicMouseSetup.exe` from the repository [Releases](https://github.com/ecarmeli/autohotkey-ergonomic-mouse/releases).
2. Double-click the installer and follow the wizard instructions.

---

## 🩺 Why This Project Exists: Vision & Objectives

* **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger/wrist strain and Repetitive Strain Injuries (RSI).
* **The Solution:** Shifting primary mouse clicks directly to the keyboard's middle Function keys (`F5`-`F7`). By separating cursor movement from clicking, this project creates a balanced, two-handed workflow that minimizes mechanical fatigue on your primary joints.

---

## 🛠️ Installation & Management

The installation wizard dynamically adapts its execution layer based on user privilege:

* **User Mode (Standard Privilege):** Installs strictly within `%LocalAppData%\ErgonomicMouse` and registers interactive logon tasks. Ideal for restricted corporate environments without administrative rights.
* **System Mode (Elevated Privilege):** Installs globally to `%ProgramData%\ErgonomicMouse` utilizing a native Windows UAC self-elevation handoff block. Installs a high-integrity Task Scheduler logon hook to ensure script functionality remains active even when interacting with administrative applications (e.g., Task Manager, elevated IDEs, or terminals).

---

## 💻 Verified Compatibility

This solution is engineered to play nicely out of the box with core enterprise productivity software and development tools:

* **Browsers:** Microsoft Edge, Google Chrome
* **Productivity:** Microsoft Office Suite
* **IDEs & Editors:** Visual Studio Code, Notepad++

---

## ✨ Key Features & Ergonomic Design

* **RSI Strain Relief:** Balances physical workload across both hands by moving high-frequency clicking tasks away from the mouse.
* **Tendon Protection:** Holding down a key triggers a single continuous mouse click instead of sending rapid, exhausting inputs.
* **Frictionless Drag-and-Drop:** An automatic 2px micro-movement triggers on initial click-hold, forcing picky applications or IDEs to register drag actions instantly without requiring a tense, heavy grip.
* **Smooth Side-Scrolling:** Use `Shift + Scroll Wheel` to pan horizontally across wide data structures, codebases, or spreadsheets seamlessly.
* **Instant Master Toggle:** Uses the physical `Scroll Lock` key (and its native hardware LED) as a global toggle to seamlessly transition between mouse mode and standard typing.
* **"Set and Forget" Automation:** A silent background engine manages seamless upgrades and a clean uninstallation automatically, ensuring zero user friction.

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

## 📋 System Requirements

* **Operating System:** Microsoft Windows 10 or Windows 11 (64-bit architecture required).
* **Execution Privileges:** * *Standard User Mode:* Does **not** require local administrator rights (installs isolated to user profile space).
  * *System-Wide Mode:* Local administrator privileges are required during setup to bind tasks directly to the elevated Task Scheduler engine.
* **Dependencies:** None. The installer bundles the precise, verified AutoHotkey v2 core binary and compiled Go launcher modules out of the box.
* **Hardware Interactivity:** Mappings take advantage of standard peripheral inputs. The Master Toggle relies on a physical `Scroll Lock` key layout; systems missing this physical key can trigger it via standard virtual keyboard overlays or alternate custom remappings.

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
│   │   └── main.go                    # Entry point binary; manages lifecycle/updates
│   └── deploymanager/ 
│       └── main.go                    # State manager engine; configures COM tasks & logs
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

---

## 🔄 Update Model (High-Level)

The application features a decoupled architecture designed for automated, zero-friction maintenance:
* **Lifecycle Orchestration:** The compiled Go launcher binary acts as the primary execution wrapper, handling background state checks and background update loops.
* **Runtime Payload:** The lightweight `ErgonomicMouse.ahk` script acts as the native hotkey payload, executed dynamically by the underlying verified AHK core engine.
* **Secure Delivery:** Updates are fetched, cryptographically/structurally validated silently in the background, and seamlessly hot-swapped upon the next user logon to ensure a resilient, frictionless experience.