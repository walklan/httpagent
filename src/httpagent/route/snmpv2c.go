package route

import (
	"config"
	"fmt"
	wsnmp "github.com/cdevr/wapsnmp"
	"httpagent/util"
	"strings"
	"time"
)

const snmpgetfail = "snmp get failed"

func Snmpv2c(seq, ip, community, oids, snmpversion string, timeout time.Duration, retry, interval, count int) SnmpResult {
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
					go Snmpv2cwalk(seq, async_c, data_c, m, snmpsess, &cacheflag)
				}
			case "get":
				for _, m := range strings.Split(mo[1], ",") {
					tasks++
					go Snmpv2cget(seq, async_c, data_c, m, snmpsess, &cacheflag)
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

func Snmpv2cwalk(seq string, async_c chan int, data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP, cflag *int) {
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

func Snmpv2cget(seq string, async_c chan int, data_c chan SnmpResult, oid string, snmp *wsnmp.WapSNMP, cflag *int) {
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
