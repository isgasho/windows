!addplugindir "."

!include StrFunc.nsh
; x64.nsh for architecture detection
!include x64.nsh
; Install for all users. MultiUser.nsh also calls SetShellVarContext to point 
; the installer to global directories (e.g. Start menu, desktop, etc.)
!define MULTIUSER_EXECUTIONLEVEL Admin
!include MultiUser.nsh
; nsProcess is used to kill running UI app.
!include nsProcess.nsh

${StrLoc}

Name NextDNS
BrandingText "NextDNS Inc"
!define VERSIONMAJOR 1
!define VERSIONMINOR 0
!define VERSIONBUILD 5
!define MUI_PRODUCT "NextDNS"
!define MUI_FILE "NextDNS"
!define MUI_VERSION "${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD}" 
!define MUI_ICON "..\resource\icon.ico"
!define MUI_UNICON "..\resource\icon.ico"
# These will be displayed by the "Click here for support information" link in "Add/Remove Programs"
# It is possible to use "mailto:" links in here to open the email client
!define HELPURL "mailto:team@nextdns.io"
!define UPDATEURL "https://nextdns.io/news"
!define ABOUTURL "https://nextdns.io"
!include MUI2.nsh

CRCCheck On
OutFile "..\NextDNSSetup-${MUI_VERSION}.exe" 
InstallDir "$PROGRAMFILES\${MUI_PRODUCT}"
;Get installation folder from registry if available
InstallDirRegKey HKCU "Software\${MUI_PRODUCT}" ""

;ShowInstDetails show
;ShowUninstDetails show

Function .onInit
  !insertmacro MULTIUSER_INIT
FunctionEnd

Function un.onInit
  !insertmacro MULTIUSER_UNINIT
FunctionEnd

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\resource\license.rtf"
!insertmacro MUI_PAGE_DIRECTORY

!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "TAP Device"
  SetOutPath "$INSTDIR\tap"

  UserInfo::GetAccountType
  Pop $0
  StrCmp $0 "Admin" isadmin
  MessageBox MB_OK "Sorry, NextDNS requires administrator permissions."
  Quit
  isadmin:

  ; TAP device files.
  ${If} ${RunningX64}
    File /r "..\tap\amd64\*"
  ${Else}
    File /r "..\tap\i386\*"
  ${EndIf}
  File "..\tap\add_tap_device.bat"

  ; ExecToStack captures both stdout and stderr from the script, in the order output.
  ; Set a (long) timeout in case the device never becomes visible to netsh.
  DetailPrint "Installing TAP device..."
  ReadEnvStr $0 COMSPEC
  nsExec::ExecToStack /timeout=180000 '$0 /c add_tap_device.bat'

  Pop $0
  Pop $1
  StrCmp $0 0 success

  ; The TAP device may have failed to install because the user did not want to
  ; install the device driver. If so:
  ;  - tell the user that they need to install the driver
  ;  - skip the Sentry report
  ;  - quit
  ;
  ; When this happens, tapinstall.exe prints an error message like this:
  ; UpdateDriverForPlugAndPlayDevices failed, GetLastError=-536870333
  ;
  ; We can use the presence of that magic number to detect this case.
  Var /GLOBAL DRIVER_FAILURE_MAGIC_NUMBER_INDEX
  ${StrLoc} $DRIVER_FAILURE_MAGIC_NUMBER_INDEX $1 "536870333" ">"

  StrCmp $DRIVER_FAILURE_MAGIC_NUMBER_INDEX "" taperror
  ; The term "device software" is the same as that used by the prompt, at least on Windows 7.
  MessageBox MB_OK "Sorry, you must install the device software in order to use NextDNS. Please try \
    running the installer again."
  Quit

  taperror:
  MessageBox MB_OK "Sorry, we could not configure your system to connect to NextDNS. Please try \
    running the installer again. $1"

  success:
  DetailPrint "TAP device installed..."
  SetOutPath "$INSTDIR"
  RMDir /r "$INSTDIR\tap"
SectionEnd

Section "NextDNS Service"
  SetOutPath "$INSTDIR"
  DetailPrint "Installing NextDNS Service..."

  ; Install service
  nsExec::ExecToLog '"${MUI_PRODUCT}Service.exe" -service stop'
  ${nsProcess::KillProcess} "${MUI_PRODUCT}Service.exe" $R0
  ${nsProcess::KillProcess} "dnsunleak.exe" $R0
  ${nsProcess::Unload}
  Sleep 5000
  ${If} ${RunningX64}
    File "/oname=${MUI_PRODUCT}Service.exe" "..\service\bin\amd64\service.exe"
  ${Else}
    File "/oname=${MUI_PRODUCT}Service.exe" "..\service\bin\i386\service.exe"
  ${EndIf}
  File "..\dnsunleak\bin\dnsunleak.exe"
  nsExec::ExecToLog /timeout=180000 '"${MUI_PRODUCT}Service.exe" -service install'
  nsExec::ExecToLog /timeout=180000 '"${MUI_PRODUCT}Service.exe" -service start'
SectionEnd
 
Section "NextDNS"
  SetOutPath "$INSTDIR"
  DetailPrint "Installing NextDNS UI..."

  DetailPrint "Stopping NextDNS..."
  ${nsProcess::KillProcess} "${MUI_PRODUCT}.exe" $R0
  ${nsProcess::Unload}
  Sleep 5000
  File "/oname=${MUI_PRODUCT}.exe" "..\gui\bin\gui.exe"
 
  ; Store installation folder
  WriteRegStr HKCU "Software\${MUI_PRODUCT}" "" $INSTDIR

  ; Run on startup
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "NextDNS" '"$INSTDIR\${MUI_PRODUCT}.exe"'

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Create desktop shortcut
  CreateShortCut "$DESKTOP\${MUI_PRODUCT}.lnk" "$INSTDIR\${MUI_PRODUCT}.exe" ""

	# Start menu
  SetShellVarContext all
  Delete "$SMPROGRAMS\${MUI_PRODUCT}.lnk"
  CreateShortCut "$SMPROGRAMS\${MUI_PRODUCT}.lnk" "$INSTDIR\${MUI_PRODUCT}.exe" "" "$INSTDIR\${MUI_PRODUCT}.exe" 0

  ; Write uninstall information to the registry
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "DisplayName" "${MUI_PRODUCT}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "UninstallString" "$INSTDIR\Uninstall.exe"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "InstallLocation" "$INSTDIR"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "DisplayIcon" "$INSTDIR\${MUI_PRODUCT}.exe"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "Publisher" "${MUI_PRODUCT}"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "HelpLink" "${HELPURL}"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "URLUpdateInfo" "${UPDATEURL}"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "URLInfoAbout" "${ABOUTURL}"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "DisplayVersion" "${MUI_VERSION}"
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "VersionMajor" ${VERSIONMAJOR}
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "VersionMinor" ${VERSIONMINOR}
	# There is no option for modifying or repairing the install
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "NoModify" 1
	WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}" "NoRepair" 1

  DetailPrint "Starting NextDNS..."
  ExecShell "" "$INSTDIR\${MUI_PRODUCT}.exe"

  # Write version file
  FileOpen $4 "$INSTDIR\version.txt" w
  FileWrite $4 "${MUI_VERSION}"
  FileClose $4
SectionEnd

Section "Uninstall"
  SetOutPath "$INSTDIR"

  DetailPrint "Stopping NextDNS UI..."
  ${nsProcess::KillProcess} "${MUI_PRODUCT}.exe" $R0

  DetailPrint "Removing NextDNS Service..."
  nsExec::ExecToLog /timeout=180000 '"${MUI_PRODUCT}Service.exe" -service stop'
  ${nsProcess::KillProcess} "${MUI_PRODUCT}Service.exe" $R0
  ${nsProcess::KillProcess} "dnsunleak.exe" $R0
  ${nsProcess::Unload}
  nsExec::ExecToLog /timeout=180000 '"${MUI_PRODUCT}Service.exe" -service uninstall'

  Sleep 1000

  RMDir /r /rebootok "$INSTDIR"
 
  ; Remove from start menu
  SetShellVarContext all
  Delete "$SMPROGRAMS\${MUI_PRODUCT}.lnk"

  ; Remove desktop link
  Delete "$DESKTOP\${MUI_PRODUCT}.lnk"

  ; Remove Uninstaller And Unistall Registry Entries
  DeleteRegKey HKEY_LOCAL_MACHINE "SOFTWARE\${MUI_PRODUCT}"
  DeleteRegKey HKEY_LOCAL_MACHINE "SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\${MUI_PRODUCT}"

  ; Remove install folder reg key
  DeleteRegKey /ifempty HKCU "Software\${MUI_PRODUCT}"
SectionEnd
