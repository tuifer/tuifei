:: Please install upx first, https://github.com/upx/upx/releases
for /f "delims=" %%i in ('dir /b /a-d /s "tuifei*"') do upx --best "%%i"
