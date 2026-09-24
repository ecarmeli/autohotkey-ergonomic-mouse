# Ergonomic Mouse Keys

A deployment-ready AutoHotkey v2 solution for ergonomic keyboard-to-mouse mapping featuring automated background update detection.

This project packages everything into a unified Windows installer executable (`ErgonomicMouseSetup.exe`) that handles clean deployments, native privilege handoffs, and complete uninstallation through the Windows Apps settings.

---

## 📥 Quick Start

1. Download the latest `ErgonomicMouseSetup.exe` from the repository [Releases](https://github.com/ecarmeli/autohotkey-ergonomic-mouse/releases).
2. Double-click the installer to launch the setup wizard.
   * *Note on SmartScreen:* The installer is not signed with a commercial Authenticode certificate. If Windows SmartScreen appears, click **More info** followed by **Run anyway** to proceed. Download releases only from this repository.
3. Follow the wizard instructions.
4. Turn **Scroll Lock** on to enable the mappings.

---

## 🎮 Keymaps & Controls

| Input | Action | Behavior |
| :--- | :--- | :--- |
| **`Scroll Lock`** | Master Toggle | Enables (LED ON) or disables (LED OFF) all mappings. |
| **`F5`** | Left Mouse Click | Supports click-and-drag holding and micro-nudging. |
| **`F6`** | Middle Mouse Click | Holds down middle click for canvas panning and hand-scrolling. |
| **`F7`** | Right Mouse Click | Supports click-and-drag holding and micro-nudging. |
| **`Shift + WheelUp`** | Horizontal Scroll Left | Sends horizontal scrolling to the application under the pointer. |
| **`Shift + WheelDown`** | Horizontal Scroll Right | Sends horizontal scrolling to the application under the pointer. |
| **`Ctrl + F12`** | Panic Release | Forces the release of all virtual mouse buttons if one becomes stuck. |

---

## 🩺 Why Ergonomic Mouse Keys?

Frequent mouse clicking can become uncomfortable during long work sessions. Ergonomic Mouse Keys moves common click actions to the keyboard, distributing input across both hands while leaving cursor movement on the mouse.

This utility is not medical equipment and does not claim to prevent or treat repetitive strain injuries. Users experiencing persistent discomfort should seek appropriate professional advice.

---

## ✨ Key Features & Ergonomic Design

* **Two-Handed Interaction:** Keeps cursor positioning on the mouse while moving frequent click actions to the keyboard.
* **Press, Hold & Release Behavior:** Holding down a mapped key triggers one continuous mouse-button hold instead of repeated clicks.
* **Frictionless Drag-and-Drop:** An automatic 2px micro-movement on initial click-hold helps applications and IDEs register drag actions consistently.
* **Smooth Side-Scrolling:** Use `Shift + Scroll Wheel` to pan horizontally across wide data structures, codebases, and spreadsheets.
* **Instant Master Toggle:** Uses the physical `Scroll Lock` key and its hardware LED as a global enable or disable control.
* **Two Installation Scopes:** Supports current-user installation without administrator rights and all-users installation with elevation.
* **Passive Update Detection:** Checks the latest published release and logs when a newer installer is available without silently replacing runtime files.

---

## 💻 Compatibility & System Requirements

This solution is designed to work with standard Windows input handling and has been used with core productivity software and development tools, including Microsoft Edge, Google Chrome, Microsoft Office, Visual Studio Code, and Notepad++.

* **Operating System:** Microsoft Windows 10 or Windows 11 (64-bit architecture required).
* **Dependencies:** None for normal use. The installer bundles the AutoHotkey runtime and compiled Go components.
* **Administrative Rights:** Required only for an **All Users** installation.
* **Hardware Interactivity:** The master toggle relies on the Windows `Scroll Lock` state. Systems without a physical Scroll Lock key can use the Windows On-Screen Keyboard or another remapping method.



---

## 🛠️ Installation Modes & Privileges
 
The installer offers two modes:
 
### Current User
 
Installs Ergonomic Mouse Keys only for your Windows account.
 
* Does not require administrator rights.
* Installs to `%LocalAppData%\ErgonomicMouse`.
* Starts automatically when you sign in.
* Is the recommended choice when only you need the application.
 
### All Users
 
Installs Ergonomic Mouse Keys for everyone who uses the computer.
 
* Requires administrator approval.
* Installs to `%ProgramData%\ErgonomicMouse`.
* Protects the installed files from modification by standard users.
* Is recommended for shared computers or when the mappings must work with elevated applications.
 
### Which Should I Choose?
 
Choose **Current User** for a personal installation without administrator rights. Choose **All Users** for a shared computer or when you need to interact with applications running as administrator.
 
> **Tip:** Double-click the installer to launch it normally. Choose **Current User** for your account, or **All Users** and approve the administrator prompt when asked.//
 

Only one installation mode can exist at a time. If Ergonomic Mouse Keys is already installed, the setup wizard will guide you through replacing it or switching installation modes.

### Uninstallation

Remove the application through the standard Windows Apps settings. The uninstaller removes the registered task, stops the matching AutoHotkey runtime, and removes installed files and operational logs.

---

## 🔐 Security Model

The project separates runtime executables from user-writable operational data and applies controls intended to reduce privilege-escalation and software-supply-chain risks.

* **Cross-Scope Protection:** The installer blocks a Current User installation when it was launched manually as Administrator. It also checks Windows user profiles for an existing user-mode installation to help prevent duplicate or orphaned deployments.
* **Hardened System Directory:** In All Users mode, runtime files are installed under `%ProgramData%\ErgonomicMouse`. The directory is hardened so standard users receive read and execute access rather than modification access.
* **Safe Log Routing:** Operational logs are routed to the interactive user's local profile rather than the protected system installation directory:

  ```text
  %LocalAppData%\ErgonomicMouse\logs\launcher.log
  %LocalAppData%\ErgonomicMouse\logs\deploymanager.log
  ```

* **Pinned Workflow Dependencies:** Repository-owned GitHub Actions are pinned to immutable commit SHAs.
* **Verified Runtime Dependency:** The AutoHotkey distribution downloaded during release builds is verified against a pinned SHA-256 digest.
* **Fail-Closed Release Gating:** Production artifacts are published only after required quality, vulnerability, repository, and malware checks succeed.

For vulnerability reporting, see [SECURITY.md](SECURITY.md).

---

## 🔄 Update & Lifecycle Model

Ergonomic Mouse Keys does not silently download or replace runtime files during startup.

`Launcher.exe` performs passive update detection only. It checks the latest published GitHub release, compares it with the installed version, and logs when a newer installer is available. Updates are delivered through the complete `ErgonomicMouseSetup.exe` installer so all components remain version-aligned.

The installer supports clean uninstallation and lifecycle management through the standard Windows Apps settings.

---

## 🤝 Contributing

Contributions are welcome.

Before starting substantial work, open an issue to discuss the proposed change and confirm that it fits the project's scope.

To contribute:

1. Fork the repository and create a focused branch from `main`.
2. Make the change and add or update tests where practical.
3. Run the relevant local validation commands.
4. Open a pull request describing the problem, the proposed change, and how it was tested.
5. Ensure all required repository checks pass before merging.

For general Go changes, run:

```powershell
go mod verify
go test ./...
go build ./...
go vet ./...
gofmt -w .
go tool staticcheck ./...
go tool govulncheck ./...
```

On Windows, the normal `go vet ./...` and `go tool staticcheck ./...` commands include the Windows-constrained `cmd/deploymanager` package. The repository CI workflow also performs explicit Windows-targeted analysis from its Linux runner.

Please keep pull requests focused and avoid unrelated formatting, refactoring, or dependency changes.

---

## 🐞 Reporting Issues

Use [GitHub Issues](https://github.com/ecarmeli/autohotkey-ergonomic-mouse/issues) for reproducible bugs, compatibility problems, and feature requests.

Include the following information where relevant:

* Windows version
* Application version
* Installation scope: Current User or All Users
* Reproduction steps
* Expected and actual behavior
* Relevant launcher, deployment-manager, or installation logs

Do not use public issues to report suspected security vulnerabilities. Follow the private reporting process described in [SECURITY.md](SECURITY.md).

---

## 📂 Repository Structure

```text
autohotkey-ergonomic-mouse/
│
├── .github/
│   ├── workflows/
│   │   ├── security-and-quality.yml        # CI: tests, builds, linting, and security scanning
│   │   ├── build-and-release.yml           # CD: Go compilation, packaging, malware scan, and release
│   │   └── monitor-autohotkey-version.yml  # Scheduled AutoHotkey security monitoring
│   └── dependabot.yml                      # Go module and GitHub Actions dependency updates
│
├── cmd/
│   ├── launcher/
│   │   ├── main.go                         # Starts AHK and performs passive update detection
│   │   └── main_test.go                    # Unit tests for deterministic version-handling logic
│   └── deploymanager/
│       └── main.go                         # Configures COM tasks, process cleanup, and system ACLs
│
├── src/
│   └── ErgonomicMouse.ahk                  # Runtime AutoHotkey source script
│
├── installer.iss                           # Inno Setup compiler configuration
├── go.mod                                  # Go module and tool definitions
├── go.sum                                  # Dependency checksums
├── LICENSE                                 # MIT License
├── SECURITY.md                             # Vulnerability-reporting policy
├── .gitignore
└── README.md
```

---

## ⚙️ Build & Development

### Prerequisites

* Go version declared in `go.mod`
* Inno Setup 6 or later
* The verified AutoHotkey runtime files under `.\bin\AutoHotkey`

### Local Validation

Run the complete Go validation set before opening a pull request:

```powershell
go mod verify
go test ./...
go build ./...
go vet ./...
gofmt -w .
go tool staticcheck ./...
go tool govulncheck ./...
```

To inspect individual launcher tests:

```powershell
go test -v ./cmd/launcher
```

### Local Go Compilation

Run the following commands from the repository root to compile both Windows executables with version and build metadata:

```powershell
# Prepare the output directory
New-Item -ItemType Directory -Path .\bin -Force | Out-Null

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

Ensure that Inno Setup 6 or later is installed and that the verified AutoHotkey runtime files are present under `.\bin\AutoHotkey`, then run:

```powershell
& "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" .\installer.iss
```

The local default installer version is a development placeholder. Production version metadata is injected by the tagged release workflow.

---

## 🛡️ CI/CD Pipeline & Security Gating

The project uses GitHub Actions with cryptographically pinned action dependencies.

### 1. Security & Quality Pipeline (`security-and-quality.yml`)

* Runs on pull requests targeting `main`, pushes to `main`, and semantic-version release tags.
* Verifies Go modules and runs launcher unit tests.
* Builds the Go packages and validates formatting.
* Runs `go vet` and Staticcheck for Linux and Windows package variants, including the Windows-constrained DeployManager package.
* Runs `govulncheck` for reachable Go dependency vulnerabilities.
* Uses Trivy to scan the repository for high and critical vulnerabilities, secrets, and configuration issues.
* Cross-compiles the Windows binaries as a dry-run validation.

CodeQL default setup separately performs semantic analysis for Go and GitHub Actions workflows. The default Go analysis follows the Linux package variant, while the repository-owned workflow provides explicit Windows-targeted analysis for DeployManager.

### 2. Build & Release Pipeline (`build-and-release.yml`)

The tagged release pipeline is divided into three stages:

* **Stage 1: Build & Package (Windows):** Validates the tag, downloads and verifies the pinned AutoHotkey distribution, injects runtime metadata, compiles both Go executables, and packages the installer with Inno Setup.
* **Stage 2: Independent Malware Scan (Linux):** Downloads the generated artifacts and scans them using **ClamAV** and **YARA**. A failed scan stops the release chain.
* **Stage 3: Conditional Production Release (Linux):** Publishes the verified installer and generated release notes only when the required upstream jobs and malware scan succeed.

### 3. Dependency Monitoring

* Dependabot monitors Go modules and GitHub Actions dependencies.
* A scheduled workflow monitors the pinned AutoHotkey release line for security-related upstream releases and advisories.

---

## 🔒 Security

Do not report suspected vulnerabilities through public issues, discussions, or pull requests. Review [SECURITY.md](SECURITY.md) and use the repository's private vulnerability-reporting process.

---

## 📄 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
