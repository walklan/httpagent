package main

import (
	"config"
	"github.com/go-martini/martini"
	"httpagent/route"
	"httpagent/util"
	"log"
	"net/http"
	"os"
	"runtime"
)

var logger *log.Logger

func main() {

	port := config.Port
	os.Setenv("PORT", port)
	os.Setenv("MARTINI_ENV", martini.Prod)
	mux := martini.Classic()
	logger = util.Applog.GetLogger()
	mux.Map(logger)

	// support get and post method
	mux.Get("/snmpagent", route.SnmpAgent)
	mux.Post("/snmpagent", route.SnmpAgent)

	// ping agent
	mux.Get("/pingagent", route.PingAgent)
	mux.Post("/pingagent", route.PingAgent)

	mux.Run()
	cpunum := runtime.NumCPU()
	//ret := runtime.GOMAXPROCS(cpunum)

	util.Info("listen port:", port, "cpunum:", cpunum)
	errs := http.ListenAndServe(":"+port, nil)
	if nil != errs {
		log.Fatalf("listen port %s error:%s", port, errs)
	}
}
