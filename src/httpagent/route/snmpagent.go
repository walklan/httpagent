package route

import (
	"config"
	"fmt"
	wsnmp "github.com/cdevr/wapsnmp"
	"httpagent/util"
	"net/http"
	"regexp"
	"strings"
	"time"
)

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
		result = Snmp(seq, ip, community, oids, version, timeout, retry, interval, count)

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
		result = SnmpResult{Error: err}
	}

	//return
	RouteJson(w, &result)
}

func Snmp(seq, ip, community, oids, snmpversion string, timeout time.Duration, retry, interval, count int) SnmpResult {
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
	snmpsess, cacheflag, err := util.SessPool.Get(ip, community, version, timeout, retry)
	defer util.SessPool.Free(ip, cacheflag)
	if err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, ip, oids, err)
		return snmpresult
	}
	if config.Debug {
		util.Debug(ip, oids, snmpsess, cacheflag)
	}

	// col more times
	for i := 0; i < count; i++ {
		// 同一个sess的请求不再goroutine, 否则可能出现错位
		async_c := make(chan int, 1)
		data_c := make(chan SnmpResult)
		tasks := 0
		for _, mib := range strings.Split(oids, "!") {
			mo := strings.Split(mib, ":")
			switch mo[0] {
			case "table":
				for _, m := range strings.Split(mo[1], ",") {
					tasks++
					go Snmpgettable(seq, async_c, data_c, m, snmpsess, &cacheflag)
				}
			case "get":
				for _, m := range strings.Split(mo[1], ",") {
					tasks++
					go Snmpget(seq, async_c, data_c, m, snmpsess, &cacheflag)
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

func Snmpgettable(seq string, async_c chan int, data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP, cflag *int) {
	async_c <- 1
	defer func() { <-async_c }()
	snmpresult := SnmpResult{Error: ""}
	table, err, retry := snmp.GetTable(util.OidPool.GetParseOid(oid))
	if err != nil || len(table) == 0 {
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, "", snmpgetfail})
		snmpresult.Error = snmpgetfail
		util.Error(seq, snmp.Target, oid, snmpresult.Error)
		data_c <- snmpresult
		return
	}
	for k, v := range table {
		if config.Debug {
			util.Debug(seq, snmp.Target, k, v)
		}
		snmpresult.Data = append(snmpresult.Data, UnitResult{k, fmt.Sprint(v), ""})
	}
	data_c <- snmpresult

	if retry > 0 {
		resetSess(snmp, cflag)
	}
}

func Snmpget(seq string, async_c chan int, data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP, cflag *int) {
	async_c <- 1
	defer func() { <-async_c }()
	snmpresult := SnmpResult{Error: ""}
	result, err, retry := snmp.Get(util.OidPool.GetParseOid(oid))
	if err != nil {
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, "", snmpgetfail})
		snmpresult.Error = snmpgetfail
		util.Error(seq, snmp.Target, oid, snmpresult.Error)
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
			util.Debug(seq, snmp.Target, oid, result)
		}
		snmpresult.Data = append(snmpresult.Data, UnitResult{oid, fmt.Sprint(result), ""})
	}
	data_c <- snmpresult

	if retry > 0 {
		if config.Debug {
			util.Debug(snmp.Target, oid, "retry:", retry, "cflag:", cflag)
		}
		resetSess(snmp, cflag)
	}
}

func resetSess(snmp *wsnmp.WapSNMP, cflag *int) {
	// udp采集重试后原session不可用，设置其为不可用，等待超时自动清理
	util.SessPool.Unavailable(snmp.Target, *cflag)
	// 重新获取新的session
	snmpsess, cacheflag, _ := util.SessPool.Get(snmp.Target, snmp.Community, snmp.Version, snmp.Timeout, snmp.Retries)
	// 修改snmpsess和cflag值
	*snmp = *snmpsess
	*cflag = cacheflag
	if config.Debug {
		util.Debug("snmp retry, reset session:", snmp.Target, snmp, "cflag:", cflag)
	}
}
