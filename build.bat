@echo off

rem Set environment variables for Windows AMD64 build
set GOOS=windows
set GOARCH=amd64
go build -o dockerpull-windows-amd64.exe main.go

rem Reset GOOS and set for Linux AMD64 build
set GOOS=linux
set GOARCH=amd64
go build -o dockerpull-linux-amd64 main.go
