package util

import (
	wsnmp "github.com/cdevr/wapsnmp"
	"sync"
)

type OidParsePool struct {
	pLock *sync.Mutex
	Oid   map[string]wsnmp.Oid
}

// 代理机采集的oid数量有限，缓存不大，先不做清理
// TODO: 设置缓存数量大小，定期清理调用次数少的oid缓存
var OidPool = &OidParsePool{Oid: make(map[string]wsnmp.Oid), pLock: &sync.Mutex{}}

func (oidpool *OidParsePool) GetParseOid(oid string) wsnmp.Oid {
	if oidparse, ok := oidpool.Oid[oid]; ok {
		return oidparse
	} else {
		result := wsnmp.MustParseOid(oid)
		oidpool.PutParseOid(oid, result)
		return result
	}
}

func (oidpool *OidParsePool) PutParseOid(oid string, oidparse wsnmp.Oid) {
	oidpool.pLock.Lock()
	defer oidpool.pLock.Unlock()
	oidpool.Oid[oid] = oidparse
}
