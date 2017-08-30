package route

import (
	"config"
	"httpagent/util"
	"net/http"
	"regexp"
	"strings"
)

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

	// v3 para
	username := r.Form.Get("username")
	securitylevel := r.Form.Get("securitylevel")
	authpass := r.Form.Get("authpass")
	authprotcol := r.Form.Get("authprotcol")
	privpass := r.Form.Get("privpass")
	privprotcol := r.Form.Get("privprotcol")

	// paramter check
	paramap := map[string]string{"seq": seq, "ip": ip, "oids": oids, "version": version}
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
		if version == "v3" {
			result = Snmpv3(seq, ip, oids, version, timeout, retry, interval, count, username, securitylevel, authpass, authprotcol, privpass, privprotcol)
		} else {
			result = Snmpv2c(seq, ip, community, oids, version, timeout, retry, interval, count)
		}

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
