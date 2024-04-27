#!/bin/sh
go build -o dockerpull main.go
# GOOS=linux GOARCH=amd64 go build -o dockerpull-linux-amd64 main.go
# GOOS=linux GOARCH=arm64 go build -o gohttpd-linux-arm64 main.go
