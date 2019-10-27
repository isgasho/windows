msbuild nextdns-windows.sln /t:Rebuild /p:Configuration=Release /p:Platform="any cpu" || goto :error

cd service

set GOARCH=amd64
C:\Go\bin\go build -o .\bin\amd64\service.exe . || goto :error

set GOARCH=386
C:\Go\bin\go build -o .\bin\i386\service.exe . || goto :error

cd ..

call :sign dnsunleak\bin\dnsunleak.exe
call :sign service\bin\amd64\service.exe
call :sign service\bin\i386\service.exe

"C:\Program Files (x86)\NSIS\makensis.exe" nsis\NextDNSSetup.nsi || goto :error

call :sign NextDNSSetup-*.exe

goto :EOF

:error
echo Failed with error #%errorlevel%.
exit /b %errorlevel%

:Sign
"C:\Program Files (x86)\Windows Kits\10\bin\x86\signtool" sign /q /n "NextDNS" /tr http://timestamp.globalsign.com/?signature=sha2 /td sha256 %~1 || goto error
exit /B 0