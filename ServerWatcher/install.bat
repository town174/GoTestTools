D:
cd %~dp0
sc create ServerWatcher binPath= "%~dp0ServerWatcher.exe" start= auto
net start ServerWatcher
pause