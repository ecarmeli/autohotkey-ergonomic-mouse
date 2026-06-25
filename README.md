# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping, featuring seamless, automated background updates.

This project features a unified, state-aware deployment manager (`InstallErgonomicMouse.bat`) that handles both installation and clean uninstallation from a single interactive menu.

1. Download or clone this repository to your local machine.
2. Double-click `InstallErgonomicMouse.bat` to launch the deployment menu.


## 🩺 Why This Project Exists: Vision & Objectives

> **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger and wrist strain.<br>
> **The Solution:** By separating cursor movement from clicking, this project creates a balanced, two-handed layout built for long-term joint health and a pain-free workflow.

This project was born out of a vital need for workspace ergonomics and physical strain relief. In a traditional computing setup, the primary mouse hand bears the brunt of continuous, repetitive stress from constant finger clicking — a leading cause of hand fatigue and repetitive strain injuries (RSI).

This solution rebalances that physical workload by shifting primary mouse click actions directly to the keyboard's middle Function keys. By utilizing your secondary hand to execute clicks, you instantly offload the mechanical stress from your mouse hand. This creates a highly efficient, two-handed workflow that minimizes finger strain and promotes a much healthier, more comfortable computing experience.


## 🛠️ Installation & Management

This project features a unified, state-aware deployment manager (`InstallErgonomicMouse.bat`) that handles both installation and clean uninstallation from a single interactive menu.

1. Download or clone this repository to your local machine.
2. Double-click `InstallErgonomicMouse.bat` to launch the deployment menu.
3. Select your preferred deployment environment:
   * **System Mode (Requires Admin):** Installs globally. Recommended if you need the script to interact seamlessly with other software running as administrator (e.g., Task Manager, elevated terminals). *Note: User Mode is recommended in a Windows Remote Desktop Services (RDS) environment.*
   * **User Mode (Standard Privilege):** Installs per-user into `$ENV:LOCALAPPDATA`. Recommended for strictly managed corporate machines where local administrative permissions are blocked.

**Smart Guardrails:** The deployment manager enforces strict mutual exclusivity. If one mode is installed, it will lock the other mode to prevent system conflicts. It also features privilege-context awareness, intentionally blocking User-Mode management if the terminal is accidentally elevated, ensuring your scheduled task permissions are never corrupted.


## 📝 Prerequisites

* **PowerShell 5.1+** (Native on Windows 10 and Windows 11).
* **AutoHotkey v2** (The installation scripts will automatically download and provision the portable engine if it is not found).


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
│   │   └── security-and-quality.yml            # CI/CD pipeline (PSScriptAnalyzer & Trivy catchall)
│   └── dependabot.yml                          # Automated weekly updates for GitHub Actions
│
├── bin/                                        # Payload directory
│   ├── ErgonomicMouse.ahk                      # Unified AutoHotkey mapping logic (System & User)
│   ├── LaunchAndUpdate.ps1                     # System-Mode smart-updater & engine
│   ├── LaunchAndUpdate-User.ps1                # User-Mode smart-updater & engine
│   ├── registerErgonomicMouseSchdTask.ps1      # System-Mode task registration
│   └── registerErgonomicMouseSchdTask-User.ps1 # User-Mode task registration
│
├── InstallErgonomicMouse.bat                   # Unified interactive deployment & cleanup manager
├── .gitignore                                  # Specifies untracked files (ignores .tmp files and logs)
└── README.md                                   # Project documentation
```
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


## ⚙️ How the Auto-Update System Works

Instead of running `ErgonomicMouse.ahk` directly, the registered Windows Scheduled Task points to the corresponding launcher engine (`LaunchAndUpdate.ps1` or `LaunchAndUpdate-User.ps1`).

1. **ETag Conditional Requests:** At logon, PowerShell opens a silent connection to GitHub. It uses native HTTP `If-None-Match` headers to compare the local file's ETag with the remote server. If the file hasn't changed, GitHub returns a `304 Not Modified`, saving bandwidth and bypassing disk I/O entirely.
2. **Payload Verification:** If an update is required, the file is downloaded into memory and scanned for specific validation strings (e.g., `#Requires AutoHotkey`) to ensure it is a complete, healthy script.
3. **Atomic File Swaps:** To prevent file corruption from sudden network drops, the new script is written to a temporary `.tmp` file, then atomically moved to overwrite the live script.
4. **Log Rotation:** A self-cleaning log tracks update statuses, automatically archiving logs over 1MB and purging archives older than 90 days.
5. **Offline Resiliency:** If you are offline, the web request fails silently within 8 seconds and immediately launches the locally cached script, ensuring your peripheral setup never stops working.