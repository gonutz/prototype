@echo off

set GOOS=windows
set GOARCH=amd64
go run .
if %errorlevel% neq 0 goto error

set GOOS=js
set GOARCH=wasm
go build -o main.wasm
if %errorlevel% neq 0 goto error

set GOOS=windows
set GOARCH=amd64
go run serve.go
if %errorlevel% neq 0 goto error

goto end

:error
echo ERROR
pause

:end
