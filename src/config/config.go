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
	"time"
)

var cfgfile string = filepath.Dir(os.Args[0]) + "/../config/httpagent.yml"

// system level config interface
var Maxsesspool int = 1000
var Maxlifetime time.Duration = 30 * time.Second
var Timeout time.Duration = 2 * time.Second
var Retry int = 1
var Debug bool = false
var Port string = "1216"

// var Logdir string = "./"
var Asyncnum int = 100

// default 100M
var Logarchsize int64 = 104857600

func init() {
	configmap, err := GetConfig()
	if err != nil {
		log.Println(err)
		return
	}
	// port
	port := GetKey("http.port", configmap)
	if port != "" {
		Port = port
	}
	// timeout
	timeout := GetKey("snmp.timeout", configmap)
	if timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			Timeout = time.Duration(t) * time.Second
		}
	}
	// retry
	retry := GetKey("snmp.retry", configmap)
	if retry != "" {
		r, err := strconv.Atoi(retry)
		if err == nil {
			Retry = r
		}
	}
	// debug
	debug := GetKey("log.debug", configmap)
	if debug == "true" || debug == "yes" {
		Debug = true
	}
	// Maxsesspool
	maxsesspool := GetKey("snmp.maxsesspool", configmap)
	if maxsesspool != "" {
		m, err := strconv.Atoi(maxsesspool)
		if err == nil {
			Maxsesspool = m
		}
	}
	// Maxlifetime
	maxlifetime := GetKey("snmp.maxlifetime", configmap)
	if maxlifetime != "" {
		t, err := strconv.Atoi(maxlifetime)
		if err == nil {
			Maxlifetime = time.Duration(t) * time.Second
		}
	}

	// Asyncnum
	asyncnum := GetKey("snmp.asyncnum", configmap)
	if asyncnum != "" {
		m, err := strconv.Atoi(asyncnum)
		if err == nil {
			Asyncnum = m
		}
	}
	// logdir
	// logdir := GetKey("log.logdir", configmap)
	// if logdir != "" {
	// 	Logdir = logdir
	// }

	// logarchsize
	logarchsize := GetKey("log.logarchsize", configmap)
	if logarchsize != "" {
		las, err := strconv.ParseInt(logarchsize, 10, 64)
		if err == nil {
			Logarchsize = las
		}
	}

	// 命令行参数
	Porttmp := flag.String("port", Port, "http listen port")
	Retrytmp := flag.Int("retry", Retry, "snmp retry times")
	Asyncnumtmp := flag.Int("asyncnum", Asyncnum, "snmp asyncnum")
	Debugtmp := flag.Bool("debug", Debug, "debug")
	// Logdirtmp := flag.String("logdir", Logdir, "log directory")
	Maxsesspooltmp := flag.Int("maxsesspool", Maxsesspool, "snmp maxsesspool")
	Logarchsizetmp := flag.Int64("logarchsize", Logarchsize, "logarchsize")
	Maxlifetimetmp := flag.Duration("maxlifetime", Maxlifetime, "snmp maxlifetime")
	Timeouttmp := flag.Duration("timeout", Timeout, "snmp timeout")
	flag.Parse()
	Port = *Porttmp
	Retry = *Retrytmp
	Asyncnum = *Asyncnumtmp
	Debug = *Debugtmp
	// Logdir = *Logdirtmp
	Maxsesspool = *Maxsesspooltmp
	Logarchsize = *Logarchsizetmp
	Maxlifetime = *Maxlifetimetmp
	Timeout = *Timeouttmp
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
