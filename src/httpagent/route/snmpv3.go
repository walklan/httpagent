package route

import (
	"config"
	"fmt"
	"github.com/k-sone/snmpgo"
	"httpagent/util"
	"strings"
	"time"
)

// v3不使用缓存池
func Snmpv3(seq, ip, oids, snmpversion string, timeout time.Duration, retry, interval, count int, username, securitylevel, authpass, authprotcol, privpass, privprotcol string) SnmpResult {
	snmpresult := SnmpResult{Error: "", Starttime: time.Now().Format("20060102150405.000")}
	if snmpversion != "v3" {
		snmpresult.Error = "unsupport snmp version(" + snmpversion + ")"
		return snmpresult
	}

	Securitylevel := getsecuritylevel(securitylevel)
	if Securitylevel < 0 {
		snmpresult.Error = "unsupport securitylevel(" + securitylevel + ")"
		return snmpresult
	}

	AuthProtcol := authProtcol(authprotcol)
	PrivProtcol := privProtcol(privprotcol)

	snmpsess, err := snmpgo.NewSNMP(snmpgo.SNMPArguments{
		Version:       snmpgo.V3,
		Address:       ip + ":161",
		Retries:       uint(retry),
		Timeout:       timeout,
		UserName:      username,
		SecurityLevel: Securitylevel,
		AuthPassword:  authpass,
		AuthProtocol:  AuthProtcol,
		PrivPassword:  privpass,
		PrivProtocol:  PrivProtcol,
	})

	if err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, ip, oids, err)
		snmpresult.Endtime = time.Now().Format("20060102150405.000")
		return snmpresult
	}

	for i := 0; i < count; i++ {
		// 同一个sess的请求不再goroutine, 否则可能出现错位
		async_c := make(chan int, 1)
		data_c := make(chan SnmpResult)
		tasks := 0
		for _, mib := range strings.Split(oids, "!") {
			mo := strings.Split(mib, ":")
			switch mo[0] {
			case "table":
				tasks++
				go Snmpv3get(seq, async_c, data_c, mo[1], snmpsess, "walk")
			case "get":
				tasks++
				go Snmpv3get(seq, async_c, data_c, mo[1], snmpsess, "get")
			default:
				// do nothing
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

func Snmpv3get(seq string, async_c chan int, data_c chan SnmpResult, oid string, snmp *snmpgo.SNMP, method string) {
	async_c <- 1
	defer func() { <-async_c }()
	snmpresult := SnmpResult{Error: ""}
	oids, err := snmpgo.NewOids(strings.Split(oid, ","))
	if err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, snmp.GetArgs().Address, oid, err)
		data_c <- snmpresult
		return
	}

	if err := snmp.Open(); err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, snmp.GetArgs().Address, oid, err)
		data_c <- snmpresult
		return
	}
	defer snmp.Close()

	var pdu snmpgo.Pdu
	if method == "get" {
		pdu, err = snmp.GetRequest(oids)
	} else if method == "walk" {
		pdu, err = snmp.GetBulkWalk(oids, 0, 1)
	} else {
		snmpresult.Error = "unsupport snmpv3 method(" + method + ")"
		util.Error(seq, snmp.GetArgs().Address, oid, snmpresult.Error)
		data_c <- snmpresult
		return
	}

	if err != nil {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, snmp.GetArgs().Address, oid, err)
		data_c <- snmpresult
		return
	}
	if pdu.ErrorStatus() != snmpgo.NoError {
		snmpresult.Error = fmt.Sprint(err)
		util.Error(seq, snmp.GetArgs().Address, oid, pdu.ErrorStatus(), pdu.ErrorIndex())
		data_c <- snmpresult
		return
	}

	// get VarBind list
	for _, v := range pdu.VarBinds() {
		if config.Debug {
			util.Debug(seq, snmp.GetArgs().Address, v.Oid, v.Variable)
		}
		snmpresult.Data = append(snmpresult.Data, UnitResult{fmt.Sprint(v.Oid), fmt.Sprint(v.Variable), ""})
	}

	data_c <- snmpresult
}

func getsecuritylevel(s string) snmpgo.SecurityLevel {
	S := strings.ToUpper(s)
	switch S {
	case "NOAUTHNOPRIV":
		return snmpgo.NoAuthNoPriv
	case "AUTHNOPRIV":
		return snmpgo.AuthNoPriv
	case "AUTHPRIV":
		return snmpgo.AuthPriv
	default:
		return -1
	}
	return -1
}

func authProtcol(p string) snmpgo.AuthProtocol {
	P := strings.ToUpper(p)
	switch P {
	case "MD5":
		return snmpgo.Md5
	case "SHA":
		return snmpgo.Sha
	}
	return ""
}

func privProtcol(p string) snmpgo.PrivProtocol {
	P := strings.ToUpper(p)
	switch P {
	case "DES":
		return snmpgo.Des
	case "AES":
		return snmpgo.Aes
	}
	return ""
}
