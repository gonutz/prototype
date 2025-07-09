@echo off
setlocal
pushd %~dp0

go run -tags=glfw .
go run .
drawsm run

popd
