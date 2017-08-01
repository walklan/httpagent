package route

import (
	"config"
	"httpagent/util"
	"httpagent/util/fastping"
	"net"
	"net/http"
	"strings"
	"time"
)

type PingResult struct {
	Data      []PingUnit
	Starttime string
	Endtime   string
	Error     string
}

type PingUnit struct {
	IP     string
	Status int
	Lag    string
}

func ParameterCheckPing(m map[string]string) string {
	err := ""
	for k, v := range m {
		if v == "" {
			err = "parameter error: '" + k + "' is null"
			break
		} else if k == "ip" {
			iplist := strings.Split(m[k], ",")
			if len(iplist) < 1 {
				err = "parameter error: ip is null"
				break
			}
		}
	}
	return err
}

func PingAgent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	seq := r.Form.Get("seq")
	ip := r.Form.Get("ip")
	retryu := r.Form.Get("retry")

	// paramter check
	paramap := map[string]string{"seq": seq, "ip": ip}
	err := ParameterCheckPing(paramap)
	util.Info(paramap)
	retry := Getretry(retryu, config.Pingretry)

	var result PingResult
	if err == "" {
		result = batchping(ip, retry)
	} else {
		result = PingResult{Error: err}
	}

	//return
	RouteJson(w, &result)
}

func batchping(ip string, pingretry int) PingResult {
	pingresult := PingResult{Error: "", Starttime: time.Now().Format("20060102150405.000")}

	pingresulttmp := doping(ip)
	pingresult.Error = pingresulttmp.Error
	pingresult.Data = pingresulttmp.Data

	for i := 0; i < pingretry; i++ {
		ippos := unreachableip(pingresult.Data)
		if len(ippos) > 0 {
			ipstr := ipjoin(ippos)
			pingtmp := doping(ipstr)
			for _, pu := range pingtmp.Data {
				if pu.Status == 1 {
					if pos, ok := ippos[pu.IP]; ok {
						pingresult.Data[pos].Status = pu.Status
						pingresult.Data[pos].Lag = pu.Lag
					}
				}
			}
		}
	}

	pingresult.Endtime = time.Now().Format("20060102150405.000")
	return pingresult
}

func ipjoin(ippos map[string]int) string {
	keys := make([]string, 0, len(ippos))
	for k, _ := range ippos {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

func unreachableip(pulist []PingUnit) map[string]int {
	ippos := make(map[string]int)
	for p, pu := range pulist {
		if pu.Status == 0 {
			ippos[pu.IP] = p
		}
	}
	return ippos
}

func doping(ips string) PingResult {
	if config.Debug {
		util.Debug("ping ip:", ips)
	}
	pingresult := PingResult{Error: ""}
	async_c := make(chan int, config.Maxconcurrency)
	data_c := make(chan PingResult)
	tasks := 0
	for _, addr := range strings.Split(ips, ",") {
		tasks++
		go PingAddr(async_c, data_c, addr)
	}
	for task_i := 0; task_i < tasks; task_i++ {
		pingtmp := <-data_c
		pingresult.Data = append(pingresult.Data, pingtmp.Data...)
		if pingresult.Error == "" {
			pingresult.Error = pingresult.Error
		}
	}

	return pingresult
}

func PingAddr(async_c chan int, data_c chan PingResult, addr string) {
	async_c <- 1
	defer func() { <-async_c }()
	status, lag := ping(addr)

	pingresult := PingResult{Error: ""}
	pingresult.Data = append(pingresult.Data, PingUnit{addr, status, lag})

	data_c <- pingresult
}

func ping(ip string) (int, string) {

	alive := 0
	lag := "-1"
	p := fastping.NewPinger()

	netProto := "ip4:icmp"
	if strings.Index(ip, ":") != -1 {
		netProto = "ip6:ipv6-icmp"
	}
	ra, err := net.ResolveIPAddr(netProto, ip)
	if err != nil {
		return 0, lag
	}

	p.AddIPAddr(ra)

	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		if config.Debug {
			util.Debug("IP Addr: " + addr.String() + " receive, RTT: " + rtt.String())
		}
		lag = rtt.String()
		alive = 1
	}

	//p.OnIdle = func() {
	//}
	//p.OnErr = func(addr *net.IPAddr, t int) {
	//	//fmt.Printf("Error %s : %d\n", addr.IP.String(), t)
	//}

	p.MaxRTT = time.Second
	err = p.Run()
	if err != nil {
		return 0, lag
	}

	//fmt.Printf("%s : %v\n", ip, p.Alive)

	return alive, lag
}
