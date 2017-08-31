package config

import (
	"flag"
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cfgfile string = filepath.Dir(os.Args[0]) + "/../config/httpagent.yml"

var Cfg = &Config{cLock: &sync.Mutex{}, Maxsesspool: 1000, Maxlifetime: 30 * time.Second, Timeout: 2 * time.Second, Retry: 1, Debug: false, Port: "1216", Maxconcurrency: 100, Logarchsize: 104857600, Daemon: false, Pingretry: 0}

type Config struct {
	cLock          *sync.Mutex
	Maxsesspool    int
	Maxlifetime    time.Duration
	Timeout        time.Duration
	Retry          int
	Debug          bool
	Port           string
	Maxconcurrency int
	Logarchsize    int64
	Pingretry      int
	Daemon         bool
	Signal         *string
}

func (c *Config) Loadconfig() {
	c.cLock.Lock()
	defer c.cLock.Unlock()
	configmap, err := GetConfig()
	if err != nil {
		log.Println(err)
		return
	}
	// port
	port := GetKey("http.port", configmap)
	if port != "" {
		c.Port = port
	}
	// timeout
	timeout := GetKey("snmp.timeout", configmap)
	if timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			c.Timeout = time.Duration(t) * time.Second
		}
	}
	// retry
	retry := GetKey("snmp.retry", configmap)
	if retry != "" {
		r, err := strconv.Atoi(retry)
		if err == nil {
			c.Retry = r
		}
	}
	// debug
	debug := GetKey("log.debug", configmap)
	if debug == "true" || debug == "yes" {
		c.Debug = true
	} else {
		c.Debug = false
	}
	// Maxsesspool
	maxsesspool := GetKey("snmp.maxsesspool", configmap)
	if maxsesspool != "" {
		m, err := strconv.Atoi(maxsesspool)
		if err == nil {
			c.Maxsesspool = m
		}
	}
	// Maxlifetime
	maxlifetime := GetKey("snmp.maxlifetime", configmap)
	if maxlifetime != "" {
		t, err := strconv.Atoi(maxlifetime)
		if err == nil {
			c.Maxlifetime = time.Duration(t) * time.Second
		}
	}

	// Maxconcurrency
	maxconcurrency := GetKey("ping.maxconcurrency", configmap)
	if maxconcurrency != "" {
		m, err := strconv.Atoi(maxconcurrency)
		if err == nil {
			c.Maxconcurrency = m
		}
	}

	// logarchsize
	logarchsize := GetKey("log.logarchsize", configmap)
	if logarchsize != "" {
		las, err := strconv.ParseInt(logarchsize, 10, 64)
		if err == nil {
			c.Logarchsize = las
		}
	}
	fmt.Printf("configs:\n\tPort = %v\n\tMaxsesspool = %v\n\tMaxlifetime = %v\n\tTimeout = %v\n\tRetry = %v\n\tDebug = %v\n\tMaxconcurrency = %v\n\tLogarchsize = %v\n\tDaemon = %v\n", c.Port, c.Maxsesspool, c.Maxlifetime, c.Timeout, c.Retry, c.Debug, c.Maxconcurrency, c.Logarchsize, c.Daemon)
}

func init() {
	Cfg.Loadconfig()

	// 命令行参数
	Porttmp := flag.String("port", Cfg.Port, "http listen port")
	Retrytmp := flag.Int("retry", Cfg.Retry, "snmp retry times")
	Debugtmp := flag.Bool("debug", Cfg.Debug, "debug")
	Maxsesspooltmp := flag.Int("maxsesspool", Cfg.Maxsesspool, "snmp maxsesspool")
	Logarchsizetmp := flag.Int64("logarchsize", Cfg.Logarchsize, "logarchsize")
	Maxlifetimetmp := flag.Duration("maxlifetime", Cfg.Maxlifetime, "snmp maxlifetime")
	Timeouttmp := flag.Duration("timeout", Cfg.Timeout, "snmp timeout")
	Cfg.Signal = flag.String("s", "", `send signal to the daemon
	quit - graceful shutdown
	stop - fast shutdown
	reload - reloading the configuration file`)
	Daemontmp := flag.Bool("daemon", Cfg.Daemon, "daemon")
	flag.Parse()
	Cfg.Port = *Porttmp
	Cfg.Retry = *Retrytmp
	Cfg.Debug = *Debugtmp
	// Logdir = *Logdirtmp
	Cfg.Maxsesspool = *Maxsesspooltmp
	Cfg.Logarchsize = *Logarchsizetmp
	Cfg.Maxlifetime = *Maxlifetimetmp
	Cfg.Timeout = *Timeouttmp
	Cfg.Daemon = *Daemontmp
}

// 需要做动态加载配置文件，放在主调程序做
func GetConfig() (m map[interface{}]interface{}, err error) {
	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return
	}
	//m = make(map[interface{}]interface{})
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return
	}
	return
}

func GetKey(key string, cfgmap map[interface{}]interface{}) string {
	keys := strings.Split(key, ".")
	if v, ok := cfgmap[keys[0]]; ok {
		switch t := v.(type) {
		case string:
			return fmt.Sprint(v)
		case interface{}:
			return GetValue(keys[1:], t)
		default:
			return ""
		}
	} else {
		return ""
	}
}

func GetValue(keys []string, intf interface{}) string {
	switch t := intf.(type) {
	case string:
		return fmt.Sprint(t)
	case interface{}:
		// fmt.Println(t)
		if v, ok := t.(map[interface{}]interface{}); ok {
			if len(keys) > 1 {
				return GetValue(keys[1:], v[keys[0]])
			} else {
				if r, ok := v[keys[0]]; ok {
					return fmt.Sprint(r)
				}
				return ""
			}
		} else if v, ok := t.([]interface{}); ok {
			r := ""
			for _, i := range v {
				r = GetValue(keys, i)
				if r != "" {
					break
				}
			}
			return r
		} else if v, ok := t.(string); ok {
			return fmt.Sprint(v)
		} else {
			return ""
		}
	default:
		return ""
	}
	return ""
}
