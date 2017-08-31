#!/usr/bin/sh

#nohup ./bin/httpagent 2>&1 >/dev/null &

if [ "x$1" == "xstart" ];then
    ./httpagent -daemon
elif [ "x$1" == "xstop" ];then
    ./httpagent -s stop
elif [ "x$1" == "xquit" ];then
    ./httpagent -s quit
elif [ "x$1" == "xreload" ];then
    ./httpagent -s reload
elif [ "x$1" == "xrestart" ];then
    ./httpagent -s stop
    sleep 1
    ./httpagent -daemon
fi
