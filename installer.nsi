; Reborn Plugin Autoinstaller - NSIS Installer Script
; Build with: makensis installer.nsi

!include "MUI2.nsh"

;-------------------------------------------------
; App metadata
;-------------------------------------------------
!define APP_NAME        "Reborn Plugin Autoinstaller"
!define APP_EXE         "reborn-plugin-autoinstaller.exe"
!define APP_VERSION     "1.0.0"
!define PUBLISHER       "Lineage Reborn Tools"
!define REG_UNINST_KEY  "Software\Microsoft\Windows\CurrentVersion\Uninstall\RebornPluginAutoinstaller"
!define REG_APP_KEY     "Software\RebornPluginAutoinstaller"

Name            "${APP_NAME}"
OutFile         "dist\RebornPluginAutoinstaller-Setup.exe"
InstallDir      "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey HKLM "${REG_APP_KEY}" "InstallDir"
RequestExecutionLevel admin

;-------------------------------------------------
; MUI settings
;-------------------------------------------------
!define MUI_ABORTWARNING
!define MUI_ICON                "resources/icon.ico"
!define MUI_UNICON              "resources/icon.ico"
!define MUI_FINISHPAGE_RUN      "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${APP_NAME}"

;-------------------------------------------------
; Pages
;-------------------------------------------------
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

;-------------------------------------------------
; Installer section
;-------------------------------------------------
Section "Install" SecInstall
    SetOutPath "$INSTDIR"

    ; Kill any running instance before replacing the exe
    nsExec::ExecToLog 'taskkill /F /IM "${APP_EXE}" /T'
    Sleep 800

    ; Install the application executable
    File "dist\${APP_EXE}"

    ; Write the uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; Store install dir in registry
    WriteRegStr HKLM "${REG_APP_KEY}" "InstallDir" "$INSTDIR"

    ; Register in Add/Remove Programs
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "DisplayName"    "${APP_NAME}"
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "DisplayIcon"     '"$INSTDIR\${APP_EXE}"'
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "Publisher"       "${PUBLISHER}"
    WriteRegStr   HKLM "${REG_UNINST_KEY}" "DisplayVersion"  "${APP_VERSION}"
    WriteRegDWORD HKLM "${REG_UNINST_KEY}" "NoModify"        1
    WriteRegDWORD HKLM "${REG_UNINST_KEY}" "NoRepair"        1

    ; Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\${APP_NAME}"
    CreateShortcut  "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"
    CreateShortcut  "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"   "$INSTDIR\uninstall.exe"

    ; Desktop shortcut
    CreateShortcut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"

SectionEnd

;-------------------------------------------------
; Uninstaller section
;-------------------------------------------------
Section "Uninstall"
    ; Kill any running instance
    nsExec::ExecToLog 'taskkill /F /IM "${APP_EXE}" /T'
    Sleep 800

    ; Remove application files
    Delete "$INSTDIR\${APP_EXE}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir  "$INSTDIR"

    ; Remove Start Menu
    Delete "$SMPROGRAMS\${APP_NAME}\*.*"
    RMDir  "$SMPROGRAMS\${APP_NAME}"

    ; Remove Desktop shortcut
    Delete "$DESKTOP\${APP_NAME}.lnk"

    ; Remove registry entries
    DeleteRegKey HKLM "${REG_UNINST_KEY}"
    DeleteRegKey HKLM "${REG_APP_KEY}"

    ; Remove auto-startup entry (HKCU, per-user)
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "RebornPluginAutoinstaller"

    ; Note: user config in %APPDATA%\RebornPluginAutoinstaller\ is preserved.
    ; Settings survive a reinstall. Users can delete the folder manually.

SectionEnd
