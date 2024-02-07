; Script generated by the Inno Setup Script Wizard.
; SEE THE DOCUMENTATION FOR DETAILS ON CREATING INNO SETUP SCRIPT FILES!

#define MyAppName "Alpine Client"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "Crystal Development, LLC"
#define MyAppURL "https://alpineclient.com/"
#define MyAppExeName "pinnacle-windows-arm64.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application. Do not use the same AppId value in installers for other applications.
; (To generate a new GUID, click Tools | Generate GUID inside the IDE.)
AppId={{273399CC-0638-4E11-84B9-F98538F73A21}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
OutputDir=D:\a\pinnacle\pinnacle\build\out
OutputBaseFilename=AlpineClientSetup-{#MyAppVersion}-ARM64
SetupIconFile=D:\a\pinnacle\pinnacle\pkg\windows\resources\alpine-client-icon.ico
Compression=zip
SolidCompression=no
WizardStyle=modern
MinVersion=6.2
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
UninstallDisplayIcon={uninstallexe}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "D:\a\pinnacle\pinnacle\bin\windows\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "D:\a\pinnacle\pinnacle\pkg\windows\resources\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "D:\a\pinnacle\pinnacle\pkg\windows\resources\alpine-client-icon.ico"; DestDir: "{app}"; Flags: ignoreversion
; NOTE: Don't use "Flags: ignoreversion" on any shared system files

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:ProgramOnTheWeb,{#MyAppName}}"; Filename: "{#MyAppURL}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\assets"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\cache"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\jre"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\libraries"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\logging"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\native_libraries"
Type: filesandordirs; Name: "{commonappdata}\.alpineclient\launcher.jar"
