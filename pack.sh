#!/usr/bin/sh
workdir=`pwd`
rm -f $workdir/bin/pid $workdir/httpagent.tar.gz $workdir/logs/httpagent.log
export GOPATH=$workdir
go build -o $workdir/bin/httpagent $workdir/src/httpagent.go
tar -czf httpagent.tar.gz ../httpagent/bin ../httpagent/config ../httpagent/logs
