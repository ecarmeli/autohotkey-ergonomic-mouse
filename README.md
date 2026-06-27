# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping, featuring seamless, automated background updates.

This project features a unified, state-aware deployment manager (`InstallErgonomicMouse.bat`) that handles both installation and clean uninstallation from a single interactive menu.

1. Download or clone this repository to your local machine.
2. Double-click `InstallErgonomicMouse.bat` to launch the deployment menu.


## 🩺 Why This Project Exists: Vision & Objectives

> **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger and wrist strain.  
> **The Solution:** By separating cursor movement from clicking, this project creates a balanced, two-handed layout built for long-term joint health and a pain-free workflow.

This project was born out of a vital need for workspace ergonomics and physical strain relief. In a traditional computing setup, the primary mouse hand bears the brunt of continuous, repetitive stress from constant finger clicking — a leading cause of hand fatigue and repetitive strain injuries (RSI).

This solution rebalances that physical workload by shifting primary mouse click actions directly to the keyboard's middle Function keys. By utilizing your secondary hand to execute clicks, you instantly offload the mechanical stress from your mouse hand. This creates a highly efficient, two-handed workflow that minimizes finger strain and promotes a much healthier, more comfortable computing experience.


## 🛠️ Installation & Management

1. Download or clone this repository to your local machine.
2. Double-click `InstallErgonomicMouse.bat` to launch the deployment menu.
3. Select your preferred deployment environment:
   * **System Mode (Requires Admin):**
     * Installs globally
     * Required for interaction with elevated apps
   * **User Mode (Standard Privilege):**
     * Installs under `LOCALAPPDATA`
     * Recommended for restricted/corporate environments

**Smart Guardrails:**
- Enforces mutual exclusivity between modes  
- Prevents privilege mismatches  
- Ensures scheduled tasks are created with correct permissions  


## 📝 Prerequisites

- **Windows 10 / 11**
- **AutoHotkey v2** *(automatically downloaded and provisioned if not present)*


## 🚀 Key Features & Ergonomic Design

* **RSI Strain Relief:** Shifts repetitive click stress from your mouse hand to the keyboard's Function row, instantly balancing the physical workload across both hands to prevent finger and wrist fatigue.
* **Tendon Protection:** Holding down a hotkey triggers just one continuous mouse click instead of sending rapid clicks.
* **Frictionless Drag-and-Drop:** An automatic 2px micro-movement triggers on initial click-hold, forcing picky applications or IDEs to register drag actions instantly without requiring a tense, heavy grip.
* **Horizontal Scrolling:** Bypasses rigid OS scroll constraints to deliver ultra-smooth side-scrolling via `Shift + Scroll Wheel` for effortless timeline or spreadsheet panning.
* **Instant Master Toggle:** Uses the physical `Scroll Lock` key (and its native LED) as a global master switch to seamlessly transition between ergonomic mouse mode and standard typing.
* **"Set and Forget" Automation:** A smart, state-aware installer sets up the environment without permission conflicts, while a silent background engine uses web standards to update itself seamlessly at logon.


## 📂 Repository Structure

```text
autohotkey-ergonomic-mouse/
│
├── .github/
│   ├── workflows/
│   │   └── security-and-quality.yml   # CI: build, lint, security scanning
│   └── dependabot.yml                 # Automated dependency updates
│
├── bin/
│   └── ErgonomicMouse.ahk             # AutoHotkey mapping logic
│
├── cmd/
│   ├── launcher/
│   │   └── main.go                    # Launcher executable (entry point)
│   └── deploymanager/ 
│       └── main.go                    # Deployment manager engine
│ 
├── InstallErgonomicMouse.bat          # Interactive installer
├── go.mod                             # Go module definition
├── go.sum                             # Dependency lock file
├── .gitignore
└── README.md
```

## ⚙️ Build & Development

### Build all Go components

`go build ./...`

### Build Windows binaries


`GOOS=windows GOARCH=amd64 go build -o bin/Launcher.exe ./cmd/launcher`  
`GOOS=windows GOARCH=amd64 go build -o bin/DeployManager.exe ./cmd/deploymanager`

***

## ✅ CI/CD Pipeline

GitHub Actions workflow:

`.github/workflows/security-and-quality.yml`

### Includes:

* ✅ `go build ./...` — compilation validation
* ✅ `go vet` — static analysis
* ✅ `staticcheck` — advanced linting
* ✅ `govulncheck` — dependency vulnerability scanning
* ✅ `trivy` — repository-level vulnerability, secret, and configuration scanning
* ✅ Windows cross-compilation

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

## 🔄 Update Model (High-Level)

* Launcher binary is responsible for execution flow
* AutoHotkey script acts as runtime payload
* Updates are fetched and validated before execution
* Designed for:
  * resilience
  * silent operation
  * zero user friction