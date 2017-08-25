package util

import (
	"config"
	"errors"
	wsnmp "github.com/cdevr/wapsnmp"
	"sync"
	"time"
)

var testoid = wsnmp.MustParseOid(".1.3.6.1.2.1.1.2.0")

var errReachMaxconn = errors.New("snmpagent: reach max snmp connections")

const MaxIdleSess = 5

type SessionPool struct {
	mLock    *sync.Mutex
	Sessions map[string]*[MaxIdleSess]*Session //默认一个ip只保存MaxIdleSess个session
}

type Session struct {
	Sess  *wsnmp.WapSNMP
	Idle  bool      // 是否空闲
	Atime time.Time // 上次使用时间
}

var SessPool = &SessionPool{Sessions: make(map[string]*[MaxIdleSess]*Session), mLock: &sync.Mutex{}}

func init() {
	var once sync.Once
	// 启动定时器清理session池
	once.Do(func() { go SessPool.cleaner(config.Maxlifetime) })
}

// 限定最大连接数
func (sp *SessionPool) Get(ip, community string, version wsnmp.SNMPVersion, tt time.Duration, rt int) (*wsnmp.WapSNMP, int, error) {
	var snmpsess *wsnmp.WapSNMP
	var err error

	if sp.PoolLen() >= config.Maxsesspool {
		return nil, -1, errReachMaxconn
	}

	snmpSess, c := sp.GetCacheSess(ip)

	if c < 0 {
		// 未从缓存中取到sess, 新建连接
		snmpsess, err = NewSess(ip, community, version, tt, rt)
		// 小于最大连接维持数，则缓存连接池
		if err == nil && sp.PoolLen() < config.Maxsesspool {
			if config.Debug {
				Debug("save snmp session:", ip)
			}
			c = sp.Save(ip, &Session{Sess: snmpsess})
		}
	} else {
		// 当前sess赋值
		snmpsess = snmpSess.Sess
	}

	return snmpsess, c, err
}

func (sp *SessionPool) PoolLen() int {
	sp.mLock.Lock()
	defer sp.mLock.Unlock()
	return len(sp.Sessions)
}

func (sp *SessionPool) GetCacheSess(ip string) (*Session, int) {
	sp.mLock.Lock()
	defer sp.mLock.Unlock()
	if sesslist, ok := sp.Sessions[ip]; ok {
		for i, snmpSess := range sesslist {
			if snmpSess != nil && snmpSess.Idle {
				snmpSess.Atime = time.Now()
				snmpSess.Idle = false
				return snmpSess, i
			}
		}
	}
	return nil, -1
}

func (sp *SessionPool) Save(ip string, sess *Session) int {
	sp.mLock.Lock()
	defer sp.mLock.Unlock()
	sess.Atime = time.Now()
	sess.Idle = false
	if sesslist, ok := sp.Sessions[ip]; ok {
		for i, _ := range sesslist {
			if sesslist[i] == nil {
				sesslist[i] = sess
				return i
			}
		}
	}
	sp.Sessions[ip] = &[MaxIdleSess]*Session{sess}
	return 0
}

func (sp *SessionPool) Free(ip string, c int) {
	if c >= 0 {
		sp.mLock.Lock()
		defer sp.mLock.Unlock()
		if seeslist, ok := sp.Sessions[ip]; ok {
			if seeslist[c] != nil {
				seeslist[c].Atime = time.Now()
				seeslist[c].Idle = true
			}
		}
	}
}

func (sp *SessionPool) Unavailable(ip string, c int) {
	if c >= 0 {
		sp.mLock.Lock()
		defer sp.mLock.Unlock()
		if seeslist, ok := sp.Sessions[ip]; ok {
			if seeslist[c] != nil {
				seeslist[c].Idle = false
			}
		}
	}
}

func (sp *SessionPool) remove(ip string, s *Session, i int) {
	// remove前需要加锁
	if config.Debug {
		Debug(s.Atime, time.Now())
		Debug("snmp session expired:", ip)
	}
	s.Sess.Close()
	sp.Sessions[ip][i] = nil
}

func (sp *SessionPool) cleaner(maxlifetime time.Duration) {
	const d = 10 * time.Second

	t := time.NewTimer(d)
	for {
		select {
		case <-t.C:
		}

		sp.mLock.Lock()
		for ip, sesslist := range sp.Sessions {
			for i, sess := range sesslist {
				if sess != nil {
					if sess.Atime.Before(time.Now().Add(-maxlifetime)) {
						sp.remove(ip, sess, i)
					} else {
						if config.Debug {
							Debug("ip:", ip, ", Idle:", sess.Idle, ", index:", i)
						}
						// 判断当前session使用计数是否在使用，若未使用则测试session是否可用(若不可用则删除)，若使用则不进行测试
						if sess.Idle && !snmptest(sess.Sess) {
							sp.remove(ip, sess, i)
						}
					}
				}
			}
		}
		sp.mLock.Unlock()

		t.Reset(d)
	}
}

func snmptest(s *wsnmp.WapSNMP) bool {
	r, err, _ := s.Get(testoid)
	if config.Debug {
		Debug("snmptest", r)
	}
	if err != nil {
		return false
	}
	return true
}

func NewSess(ip, community string, version wsnmp.SNMPVersion, tt time.Duration, rt int) (*wsnmp.WapSNMP, error) {
	snmpsess, err := wsnmp.NewWapSNMP(ip, community, version, tt, rt)
	Info("create snmp session:", ip)
	return snmpsess, err
}
