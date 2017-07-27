#!/usr/bin/sh
workdir=`pwd`
export GOPATH=$workdir
go build -o $workdir/bin/httpagent $workdir/src/httpagent.go
tar -czf httpagent.tar.gz ../httpagent/bin ../httpagent/config ../httpagent/logs
