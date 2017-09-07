package util

import (
	"config"
	"github.com/k-sone/snmpgo"
	"sync"
)

type OidParsePool struct {
	pLock *sync.Mutex
	Oid   map[string]*snmpgo.Oid
}

// 代理机采集的oid数量有限，缓存不大，先不做清理
// TODO: 设置缓存数量大小，定期清理调用次数少的oid缓存
var OidPool = &OidParsePool{Oid: make(map[string]*snmpgo.Oid), pLock: &sync.Mutex{}}

func (oidpool *OidParsePool) GetParseOids(s []string) (oids snmpgo.Oids, err error) {
	for _, oid := range s {
		oidparse, e := oidpool.GetParseOid(oid)
		if e != nil {
			return nil, e
		}
		oids = append(oids, oidparse)
	}
	return
}

func (oidpool *OidParsePool) GetParseOid(oid string) (*snmpgo.Oid, error) {
	if oidparse, ok := oidpool.Oid[oid]; ok {
		if config.Cfg.Debug {
			Debug("get parsed oid from cache:", oid)
		}
		return oidparse, nil
	}
	o, e := snmpgo.NewOid(oid)
	if e != nil {
		return nil, e
	}
	oidpool.PutParseOid(oid, o)
	return o, nil
}

func (oidpool *OidParsePool) PutParseOid(oid string, oidparse *snmpgo.Oid) {
	oidpool.pLock.Lock()
	defer oidpool.pLock.Unlock()
	oidpool.Oid[oid] = oidparse
}
