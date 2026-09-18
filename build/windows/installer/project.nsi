Unicode true

!define REQUEST_EXECUTION_LEVEL "user"
!define INFO_PROJECTNAME "moonman-release"
!define INFO_COMPANYNAME "Moonman Release"
!define INFO_PRODUCTNAME "Moonman Release"
!define INFO_PRODUCTVERSION "0.1.0"
!define INFO_COPYRIGHT "Moonman Release"
!define PRODUCT_EXECUTABLE "moonman-release.exe"

!include "wails_tools.nsh"
!include "MUI.nsh"
!include "LogicLib.nsh"
!include "StrFunc.nsh"
!include "WinMessages.nsh"

${Using:StrFunc} StrStr
${Using:StrFunc} StrRep

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"

ManifestDPIAware true
!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\Programs\MoonmanRelease"
ShowInstDetails show

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

Function AddUserPath
    ReadRegStr $0 HKCU "Environment" "Path"
    ${StrStr} $1 $0 "$INSTDIR"
    ${If} $1 == ""
        ${If} $0 == ""
            WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR"
        ${Else}
            WriteRegExpandStr HKCU "Environment" "Path" "$0;$INSTDIR"
        ${EndIf}
    ${EndIf}
    SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
FunctionEnd

Function un.RemoveUserPath
    ReadRegStr $0 HKCU "Environment" "Path"
    ${StrRep} $0 $0 ";$INSTDIR" ""
    ${StrRep} $0 $0 "$INSTDIR;" ""
    ${StrRep} $0 $0 "$INSTDIR" ""
    WriteRegExpandStr HKCU "Environment" "Path" "$0"
    SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
FunctionEnd

Function WriteMoonmanUninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    SetRegView 64
    WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
FunctionEnd

Function un.DeleteMoonmanUninstaller
    Delete "$INSTDIR\uninstall.exe"
    SetRegView 64
    DeleteRegKey HKCU "${UNINST_KEY}"
FunctionEnd

Section "Moonman Release" MainSection
    SectionIn RO
    !insertmacro wails.setShellContext
    SetOutPath $INSTDIR
    !insertmacro wails.files
    FileOpen $0 "$INSTDIR\moonman-release.installed" w
    FileWrite $0 "installed$\r$\n"
    FileClose $0
    CreateDirectory "$SMPROGRAMS\Moonman Release"
    CreateShortcut "$SMPROGRAMS\Moonman Release\Moonman Release.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    Call AddUserPath
    Call WriteMoonmanUninstaller
SectionEnd

Section "Desktop shortcut" DesktopSection
    CreateShortcut "$DESKTOP\Moonman Release.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext
    Call un.RemoveUserPath
    Delete "$DESKTOP\Moonman Release.lnk"
    Delete "$SMPROGRAMS\Moonman Release\Moonman Release.lnk"
    RMDir "$SMPROGRAMS\Moonman Release"
    Delete "$INSTDIR\moonman-release.installed"
    RMDir /r "$INSTDIR"
    Call un.DeleteMoonmanUninstaller
    ; User projects, logs, activity, and releases live in AppData and are kept.
SectionEnd
