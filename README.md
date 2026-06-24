# autohotkey-ergonomic-mouse

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping, featuring seamless, automated background updates.

> **The Problem:** Traditional navigation forces your mouse hand to handle positioning, scrolling, and thousands of repetitive clicks daily, causing chronic finger and wrist strain.<br>
> **The Solution:** By separating cursor movement from clicking, this project creates a balanced, two-handed layout built for long-term joint health and a pain-free workflow.

This project was born out of a vital need for workspace ergonomics and physical strain relief. In a traditional computing setup, the primary mouse hand bears the brunt of continuous, repetitive stress from constant finger clicking — a leading cause of hand fatigue and repetitive strain injuries (RSI).

This solution rebalances that physical workload by shifting primary mouse click actions directly to the keyboard's middle Function keys. By utilizing your secondary hand to execute clicks, you instantly offload the mechanical stress from your mouse hand. This creates a highly efficient, two-handed workflow that minimizes finger strain and promotes a much healthier, more comfortable computing experience.

---

## 🛠️ Installation & Deployment

### Method A: Admin Mode (Recommended)
Use this method if you have administrative privileges on the target machine and want the script to interact seamlessly with other software running as administrator (e.g., Task Manager, elevated terminals). **Note**: Method B is recommended in a Windows Remote Desktop Services (RDS) environment.

1. Download or clone this repository to the local machine.
2. Right-click `DeployErgonomicMouse.bat` and select **Run as Administrator**.
3. The installation script will create the target directory, provision `LaunchAndUpdater.ps1`, copy `ErgonomicMouse.ahk`, and register a Windows Scheduled Task to run silently at every user logon.

### Method B: User Mode (No UAC Requirements)
Use this option on strictly managed corporate machines where local administrative permissions are blocked.

1. Download or clone this repository to the local machine.
2. Double-click `DeployErgonomicMouse-User.bat` (No UAC prompt will appear).
3. The solution installs itself entirely inside the user's `$ENV:LOCALAPPDATA` directory and registers a standard-privilege user Scheduled Task to execute silently at login.

---

## 📝 Prerequisites

*   **PowerShell 5.1+** (Native on Windows 10 and Windows 11).

---

## 🚀 Key Features

*   **Anti-Repeat Logic:** Prevents Windows keyboard auto-repeat from triggering multiple rapid mouse clicks when holding down a hotkey.
*   **Micro-Nudges:** Sends a micro-movement (default 2px) on initial click to force legacy or picky applications to register drag-and-drop operations immediately.
*   **High-Precision Horizontal Scrolling:** Implements direct Win32 API `PostMessage` mechanics for ultra-smooth side-scrolling using `Shift + Scroll Wheel`.
*   **Master Hardware Toggle:** Use the `Scroll Lock` key to physically enable or disable the keyboard remapping globally on the fly.
*   **Self-Healing Auto-Updates:** A background PowerShell engine fetches the latest `.ahk` logic on system startup, with an automatic offline fallback to the local cached script if internet connectivity is missing.

---

## 📂 Repository Structure

```text
autohotkey-ergonomic-mouse/
│
├── .github/
│   ├── workflows/
│   │   └── security-and-quality.yml            # CI/CD pipeline (PSScriptAnalyzer & Trivy catchall)
│   └── dependabot.yml                          # Automated weekly updates for GitHub Actions
│
├── System Mode (UAC support)/                  # Machine-wide deployment (requires Admin elevation)
│   ├── DeployErgonomicMouse.bat                # One-click Admin installation wrapper
│   ├── ErgonomicMouse.ahk                      # Admin mode script (targets Public Documents)
│   ├── LaunchAndUpdate.ps1                     # Cryptographic smart-updater & background engine
│   └── registerErgonomicMouseSchdTask.ps1      # Scheduled Task registration installer
│
├── User Mode (no UAC support)/                 # Per-user deployment (standard non-admin privileges)
│   ├── DeployErgonomicMouse-User.bat           # One-click User installation wrapper
│   ├── ErgonomicMouse-User.ahk                 # User mode script (targets LocalAppData)
│   ├── LaunchAndUpdate-User.ps1                # Cryptographic smart-updater & background engine
│   └── registerErgonomicMouseSchdTask-User.ps1 # Scheduled Task registration installer
│
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

---

## ⚙️ How the Auto-Update System Works

Instead of running `ErgonomicMouse.ahk` directly, the Windows Scheduled Task points to `LaunchAndUpdater.ps1`. 

1. **Check & Download:** At logon, PowerShell opens a silent connection to the `raw.githubusercontent.com` path for this repository.
2. **Sanity Verification:** It downloads the new `.ahk` file to a temporary directory and verifies that it is a healthy, valid file (preventing corruption or partial downloads from breaking your mouse control).
3. **Hot-Swap:** It overwrites the local copy of `ErgonomicMouse.ahk`.
4. **Execution:** It spins up the AutoHotkey executable to load the freshly updated script safely.
5. **Offline Resiliency:** If you are offline (e.g., on an airplane or disconnected from corporate network infrastructure), the script fails silently within 8 seconds and immediately launches the local cached version so your peripheral setup never stops working.