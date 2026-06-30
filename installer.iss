[Setup]
AppId={{E8C2B3C5-9A1A-4D7E-8F4C-9C2B3D4E5F6A}
AppName=Ergonomic Mouse Keys
AppVersion=1.0.0
AppPublisher=Erez Carmeli
DefaultDirName={code:GetInstallDir}
DefaultGroupName=Ergonomic Mouse Keys
OutputDir=.\bin
OutputBaseFilename=ErgonomicMouseSetup
UsePreviousPrivileges=no
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
DisableDirPage=yes
; Enable native master logging for the entire deployment pipeline
SetupLogging=yes

[Messages]
PrivilegesRequiredOverrideTitle=Installation Mode
PrivilegesRequiredOverrideInstruction=Choose how Ergonomic Mouse Keys should be installed.
ConfirmUninstall=Are you sure you want to completely remove Ergonomic Mouse Keys and all background tasks?
DirExists=The folder: %n%n%1%n%nalready exists. Would you like to overwrite the existing configuration?
; Native overrides for the final uninstall result messages
UninstalledAll=Ergonomic Mouse Keys was successfully removed from your computer.
UninstalledMost=Uninstallation completed with warnings. Some files or background tasks could not be removed.

[Files]
Source: ".\bin\Launcher.exe"; DestDir: "{code:GetInstallDir}"; Flags: ignoreversion
Source: ".\src\ErgonomicMouse.ahk"; DestDir: "{code:GetInstallDir}"; Flags: ignoreversion
Source: ".\bin\DeployManager.exe"; DestDir: "{code:GetInstallDir}"; Flags: ignoreversion
Source: ".\bin\AutoHotkey\*"; DestDir: "{code:GetInstallDir}\AutoHotkey"; Flags: ignoreversion recursesubdirs createallsubdirs

[Run]
Filename: "{code:GetInstallDir}\DeployManager.exe"; Parameters: "--mode=system --install"; Flags: runhidden; Check: IsAdminInstallMode
Filename: "{code:GetInstallDir}\DeployManager.exe"; Parameters: "--mode=user --install"; Flags: runhidden; Check: not IsAdminInstallMode
; Run the correct scheduled task based on the user's selected installation mode
; A single entry that resolves the correct Task name at runtime
Filename: "schtasks"; Parameters: "/Run /TN ""{code:GetTaskName}"""; Flags: runhidden postinstall skipifsilent runascurrentuser; Description: "Start Ergonomic Mouse Keys"

[UninstallRun]
; Kill active processes to prevent file-lock "Access Denied" errors during uninstallation
Filename: "{cmd}"; Parameters: "/C taskkill /F /IM AutoHotkey64.exe /IM Launcher.exe"; Flags: runhidden; RunOnceId: "KillEngineProcesses"
Filename: "{code:GetInstallDir}\DeployManager.exe"; Parameters: "--mode=system --uninstall"; Flags: runhidden; RunOnceId: "TeardownSystem"; Check: IsAdminInstallMode
Filename: "{code:GetInstallDir}\DeployManager.exe"; Parameters: "--mode=user --uninstall"; Flags: runhidden; RunOnceId: "TeardownUser"; Check: not IsAdminInstallMode

[UninstallDelete]
Type: filesandordirs; Name: "{code:GetInstallDir}\logs"
Type: files; Name: "{code:GetInstallDir}\*.etag"
Type: files; Name: "{code:GetInstallDir}\*.tmp"
Type: dirifempty; Name: "{code:GetInstallDir}"

[Code]

const
  InstallModeNone = 0;
  InstallModeUser = 1;
  InstallModeSystem = 2;

function GetInstalledMode(): Integer;
var
  sUnInstPath: String;
  sDummy: String;
begin
  sUnInstPath := 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{E8C2B3C5-9A1A-4D7E-8F4C-9C2B3D4E5F6A}_is1';

  if RegQueryStringValue(HKLM, sUnInstPath, 'UninstallString', sDummy) then
  begin
    Result := InstallModeSystem;
    Exit;
  end;

  if RegQueryStringValue(HKCU, sUnInstPath, 'UninstallString', sDummy) then
  begin
    Result := InstallModeUser;
    Exit;
  end;

  Result := InstallModeNone;
end;

// Helper to find existing uninstall keys in both HKLM (System) and HKCU (User)
function GetUninstallString: String;
var
  sUnInstPath: String;
begin
  sUnInstPath :=
    'Software\Microsoft\Windows\CurrentVersion\Uninstall\{E8C2B3C5-9A1A-4D7E-8F4C-9C2B3D4E5F6A}_is1';

  Result := '';

  if RegQueryStringValue(HKLM, sUnInstPath, 'UninstallString', Result) then
    Exit;

  if RegQueryStringValue(HKCU, sUnInstPath, 'UninstallString', Result) then
    Exit;
end;

function InitializeSetup(): Boolean;
var
  V: Integer;
  ExistingMode: Integer;
  sUnInstallString: String;
  iResultCode: Integer;
begin
  Result := True;
  ExistingMode := GetInstalledMode();
  sUnInstallString := GetUninstallString();

  if ExistingMode = InstallModeNone then Exit;

  V := MsgBox(
    'Ergonomic Mouse Keys is already installed.' + #13#10 + #13#10 +
    'Would you like to uninstall the existing version before continuing?',
    mbInformation,
    MB_YESNO
  );

  if V <> IDYES then
  begin
    MsgBox('Please uninstall the existing version manually before running this setup.', mbError, MB_OK);
    Result := False;
    Exit;
  end;

  if sUnInstallString <> '' then
  begin
    // User consented. Run the uninstaller completely hidden so it doesn't steal focus or create a taskbar icon.
    if not Exec('>', sUnInstallString + ' /VERYSILENT /SUPPRESSMSGBOXES /NORESTART', '', SW_HIDE, ewWaitUntilTerminated, iResultCode) then
    begin
      MsgBox('Uninstallation failed: ' + SysErrorMessage(iResultCode), mbError, MB_OK);
      Result := False;
      Exit;
    end;
  end;
end; 

function GetInstallDir(Param: String): String;
begin
  if IsUninstaller then
    Result := ExtractFileDir(ExpandConstant('{uninstallexe}'))
  else if IsAdminInstallMode then
    Result := ExpandConstant('{commonappdata}\ErgonomicMouse')
  else
    Result := ExpandConstant('{localappdata}\ErgonomicMouse');
end;

function GetTaskName(Param: String): String;
begin
  if IsAdminInstallMode then
    Result := 'ErgonomicMouseMapping'
  else
    Result := 'ErgonomicMouseMapping-User';
end;

// Centralized Logging Handoff
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssDone then
  begin
    ForceDirectories(GetInstallDir('') + '\logs');
    CopyFile(ExpandConstant('{log}'), GetInstallDir('') + '\logs\install.log', False);
  end;
end;