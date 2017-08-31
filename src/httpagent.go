package main

import (
	"config"
	"github.com/go-martini/martini"
	"github.com/sevlyar/go-daemon"
	"httpagent/route"
	"httpagent/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

var logger *log.Logger

func main() {
	daemon.AddCommand(daemon.StringFlag(config.Cfg.Signal, "quit"), syscall.SIGQUIT, termHandler)
	daemon.AddCommand(daemon.StringFlag(config.Cfg.Signal, "stop"), syscall.SIGTERM, termHandler)
	daemon.AddCommand(daemon.StringFlag(config.Cfg.Signal, "reload"), syscall.SIGHUP, reloadHandler)

	cntxt := &daemon.Context{
		PidFileName: filepath.Dir(os.Args[0]) + "/../bin/pid",
		PidFilePerm: 0644,
		WorkDir:     "./",
	}

	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatalln("Unable send signal to the daemon:", err)
		}
		daemon.SendCommands(d)
		return
	}

	if config.Cfg.Daemon {
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatalln(err)
		}
		if d != nil {
			return
		}
	}
	defer cntxt.Release()
	go httpservice()

	// server for signal, main returned if register signal recieved
	err := daemon.ServeSignals()
	if err != nil {
		log.Println("Error:", err)
	}
}

func httpservice() {
	port := config.Cfg.Port
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

func termHandler(sig os.Signal) error {
	log.Println("terminating by signal...")
	return daemon.ErrStop
}

func reloadHandler(sig os.Signal) error {
	log.Println("configuration reloaded...")
	config.Cfg.Loadconfig()
	log.Printf("configs:\n\tPort = %v\n\tMaxsesspool = %v\n\tMaxlifetime = %v\n\tTimeout = %v\n\tRetry = %v\n\tDebug = %v\n\tMaxconcurrency = %v\n\tLogarchsize = %v\n", config.Cfg.Port, config.Cfg.Maxsesspool, config.Cfg.Maxlifetime, config.Cfg.Timeout, config.Cfg.Retry, config.Cfg.Debug, config.Cfg.Maxconcurrency, config.Cfg.Logarchsize)
	return nil
}
