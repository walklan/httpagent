#!/usr/bin/sh

#nohup ./bin/httpagent 2>&1 >/dev/null &
workdir=`dirname $0`

if [ "x$1" == "xstart" ];then
    echo "httpagent service start..."
    $workdir/httpagent -daemon
elif [ "x$1" == "xstop" ];then
    $workdir/httpagent -s stop
    echo "httpagent service stop"
elif [ "x$1" == "xquit" ];then
    $workdir/httpagent -s quit
    echo "httpagent service quit"
elif [ "x$1" == "xreload" ];then
    echo "httpagent service reload configuration"
    $workdir/httpagent -s reload
elif [ "x$1" == "xrestart" ];then
    $workdir/httpagent -s stop
    echo "httpagent service stop"
    sleep 1
    echo "httpagent service start..."
    $workdir/httpagent -daemon
fi
