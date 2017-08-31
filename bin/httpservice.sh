#!/usr/bin/sh

#nohup ./bin/httpagent 2>&1 >/dev/null &

if [ "x$1" == "xstart" ];then
    echo "httpagent service start..."
    ./httpagent -daemon
elif [ "x$1" == "xstop" ];then
    ./httpagent -s stop
    echo "httpagent service stop"
elif [ "x$1" == "xquit" ];then
    ./httpagent -s quit
    echo "httpagent service quit"
elif [ "x$1" == "xreload" ];then
    echo "httpagent service reload configuration"
    ./httpagent -s reload
elif [ "x$1" == "xrestart" ];then
    ./httpagent -s stop
    echo "httpagent service stop"
    sleep 1
    echo "httpagent service start..."
    ./httpagent -daemon
fi
