# Security Policy

## Supported Versions

Only the latest published release is supported with security updates. If you are using an earlier release, please update to the latest version before reporting an issue.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| Earlier releases | No |

## Reporting a Vulnerability

Please do not report suspected security vulnerabilities through public GitHub issues, discussions, or pull requests.

Use GitHub's private vulnerability reporting feature instead:

[Report a vulnerability privately](https://github.com/ecarmeli/autohotkey-ergonomic-mouse/security/advisories/new)

Please include as much of the following information as possible:

- A clear description of the vulnerability and its potential impact
- The affected release version and component
- Whether the installation is per-user or all-users
- Reproduction steps or proof-of-concept details
- Relevant configuration, logs, screenshots, or error messages
- Any suggested remediation, if available

Reports will be reviewed privately. If the issue is accepted, remediation and disclosure will be coordinated before details are made public. Reporter credit will be included when appropriate unless anonymity is requested.

## Security Scope

Examples of issues that are in scope include:

- Privilege escalation involving the installer, Launcher, DeployManager, scheduled tasks, or installed directories
- Unauthorized modification or execution of the AutoHotkey runtime or script
- Incorrect access control or permissions applied during installation
- Unsafe installation-scope detection or behavior across Windows user profiles
- Vulnerabilities in the update-check mechanism
- Integrity weaknesses in the build, dependency, artifact-scanning, or release process
- Exposed credentials, tokens, or other sensitive information committed to the repository

The following are generally out of scope:

- Windows SmartScreen warnings caused solely by the installer not being signed with a commercial Authenticode certificate
- Vulnerabilities that require administrator access already controlled by the attacker
- Social-engineering reports without a demonstrated technical vulnerability
- Denial-of-service reports that do not have a meaningful security impact
- Vulnerabilities in Windows, AutoHotkey, Go, or another upstream dependency that cannot be addressed in this repository; however, reports identifying an affected dependency version used by this project are welcome
- The intended keyboard-to-mouse remapping behavior of the application

## Disclosure

Please allow a security fix to be prepared and released before publicly disclosing vulnerability details. Once a fix is available, a GitHub security advisory may be published with affected versions, remediation guidance, and reporter credit where applicable.
