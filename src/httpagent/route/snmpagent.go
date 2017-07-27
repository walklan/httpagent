package route

import (
	"config"
	"fmt"
	wsnmp "github.com/cdevr/wapsnmp"
	"httpagent/util"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var async_c = make(chan int, config.Asyncnum)

const snmpgetfail = "snmp get failed"

type SnmpResult struct {
	Data      []UnitResult
	Starttime string
	Endtime   string
	Error     string
}

type UnitResult struct {
	Index string
	Value string
	Error string
}

type SnmpResultMulti struct {
	Data      []UnitResultMulti
	Starttime string
	Endtime   string
	Error     string
}

type UnitResultMulti struct {
	Index string
	Value []string
	Error string
}

func ParameterCheck(m map[string]string) string {
	err := ""
	for k, v := range m {
		if v == "" {
			err = "parameter error: '" + k + "' is null"
			break
		} else if k == "oids" {
			// snmpmethod check
			for _, mib := range strings.Split(m[k], "!") {
				m := strings.Split(mib, ":")
				if m[0] != "table" && m[0] != "get" {
					err = "parameter error: unsupport snmp method '" + m[0] + "'"
					break
				} else if len(m) > 1 {
					if match, _ := regexp.MatchString(`^[\.\d,]+$`, m[1]); !match {
						err = "parameter error: snmp oid(" + m[1] + ") format error"
						break
					}
				}
			}
		}
	}
	return err
}

func ParameterError(err string) SnmpResult {
	sr := SnmpResult{Error: err}
	return sr
}

func SnmpAgent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	seq := r.Form.Get("seq")
	ip := r.Form.Get("ip")
	community := r.Form.Get("community")
	oids := r.Form.Get("oids")
	version := r.Form.Get("version")
	timeoutu := r.Form.Get("timeout")
	retryu := r.Form.Get("retry")
	intervalu := r.Form.Get("interval")
	countu := r.Form.Get("count")

	// paramter check
	paramap := map[string]string{"seq": seq, "ip": ip, "community": community, "oids": oids, "version": version}
	err := ParameterCheck(paramap)
	util.Info(paramap)

	var result SnmpResult
	if err == "" {
		timeout := Gettimeout(timeoutu, config.Timeout)
		retry := Getretry(retryu, config.Retry)
		interval := GetparaStrtoInt(intervalu, 1)
		count := GetparaStrtoInt(countu, 1)
		if count <= 1 {
			interval = 0
		}
		// log.Println(timeoutu, config.Timeout, retryu, config.Retry, timeout, retry)
		result = Snmp(ip, community, oids, version, timeout, retry, interval, count)

		if count > 1 {
			pos := 0
			var retmap = make(map[string]int)
			var resultmulti SnmpResultMulti
			resultmulti.Error = result.Error
			resultmulti.Starttime = result.Starttime
			resultmulti.Endtime = result.Endtime
			for _, v := range result.Data {
				if valuepos, ok := retmap[v.Index]; ok {
					resultmulti.Data[valuepos].Value = append(resultmulti.Data[valuepos].Value, v.Value)
				} else {
					resultmulti.Data = append(resultmulti.Data, UnitResultMulti{v.Index, []string{v.Value}, v.Error})
					retmap[v.Index] = pos
					pos++
				}
			}
			RouteJson(w, &resultmulti)
			return
		}
	} else {
		result = ParameterError(err)
	}

	//return
	RouteJson(w, &result)
}

func GetparaStrtoInt(s string, t int) int {
	if s == "" {
		return t
	}
	m, errs := strconv.Atoi(s)
	if errs != nil {
		return t
	}
	return m
}

func Gettimeout(tu string, ts time.Duration) time.Duration {
	// if tu == "", return system level timeout
	if tu == "" {
		return ts
	}
	t, errs := strconv.Atoi(tu)
	if errs != nil {
		return ts
	}
	return time.Duration(t) * time.Second
}

func Getretry(ru string, rs int) int {
	// if ru == "", return system level retry
	if ru == "" {
		return rs
	}
	r, errs := strconv.Atoi(ru)
	if errs != nil {
		return rs
	}
	return r
}

func Snmp(ip, community, oids, snmpversion string, timeout time.Duration, retry, interval, count int) SnmpResult {
	snmpresult := SnmpResult{Error: "", Starttime: time.Now().Format("20060102150405.000")}
	version := wsnmp.SNMPv2c
	if snmpversion == "v1" {
		version = wsnmp.SNMPv1
	} else if snmpversion == "v2c" {
		version = wsnmp.SNMPv2c
	} else if snmpversion == "v3" {
		//version = wsnmp.SNMPv3
		// user,pass, ...to do
		snmpresult.Error = "unsupport snmp version(" + snmpversion + "), need to do"
		return snmpresult
	} else {
		snmpresult.Error = "unsupport snmp version(" + snmpversion + ")"
		return snmpresult
	}

	// get snmp session from pool
	snmpsess, err := util.SnmpSession.GetSession(ip, community, version, timeout, retry)
	defer util.SnmpSession.DelUsingcnt(ip)
	if err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(err)
		return snmpresult
	}

	// col more times
	for i := 0; i < count; i++ {
		// snmp goroutine
		data_c := make(chan SnmpResult)
		tasks := 0
		for _, mib := range strings.Split(oids, "!") {
			mo := strings.Split(mib, ":")
			switch mo[0] {
			case "table":
				for _, m := range strings.Split(mo[1], ",") {
					tasks++
					go Snmpgettable(data_c, m, snmpsess)
				}
			case "get":
				for _, m := range strings.Split(mo[1], ",") {
					tasks++
					go Snmpget(data_c, m, snmpsess)
				}
			default:
				// do nothing, because parameter check have been checked before
			}
		}
		for task_i := 0; task_i < tasks; task_i++ {
			snmprtmp := <-data_c
			snmpresult.Data = append(snmpresult.Data, snmprtmp.Data...)
			if snmpresult.Error == "" {
				snmpresult.Error = snmprtmp.Error
			}
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}

	snmpresult.Endtime = time.Now().Format("20060102150405.000")
	return snmpresult
}

func Snmpgettable(data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP) {
	async_c <- 1
	defer func() { <-async_c }()
	snmpresult := SnmpResult{Error: ""}
	table, err := snmp.GetTable(wsnmp.MustParseOid(oid))
	if err != nil || len(table) == 0 {
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, "", snmpgetfail})
		snmpresult.Error = snmpgetfail
		util.Error(oid, snmpresult.Error)
		data_c <- snmpresult
		return
	}
	for k, v := range table {
		if config.Debug {
			util.Debug(k, v)
		}
		snmpresult.Data = append(snmpresult.Data, UnitResult{k, fmt.Sprint(v), ""})
	}
	data_c <- snmpresult
}

func Snmpget(data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP) {
	async_c <- 1
	defer func() { <-async_c }()
	snmpresult := SnmpResult{Error: ""}
	result, err := snmp.Get(wsnmp.MustParseOid(oid))
	if err != nil {
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, "", snmpgetfail})
		snmpresult.Error = snmpgetfail
		util.Error(oid, snmpresult.Error)
		data_c <- snmpresult
		return
	}
	switch result.(type) {
	case wsnmp.UnsupportedBerType:
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, "", snmpgetfail})
		snmpresult.Error = snmpgetfail
		data_c <- snmpresult
		return
	default:
		if config.Debug {
			util.Debug(oid, result)
		}
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, fmt.Sprint(result), ""})
	}
	data_c <- snmpresult
}
