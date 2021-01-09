package uv

import (
	"github.com/zngw/count/sdb"
	"github.com/zngw/set"
	"sync"
	"time"
)

type Info struct {
	Update *set.Set // 需要更新的ip
	IP     *set.Set // IP信息
}

var UserUV sync.Map

func Init(User []string) {
	for _, u := range User {
		// 从数据库中读取
		_ = sdb.CreateUVTable(u)

		info := Info{}
		info.Update = set.New()
		info.IP = sdb.GetUVIPList(u)

		UserUV.Store(u, info)
	}

	go func() {
		for true {
			save()
			time.Sleep(time.Second)
		}
	}()
}

func Add(name, ip string) (count int) {
	if v, ok := UserUV.Load(name); ok {
		info := v.(Info)
		if info.Update.Has(ip) || info.IP.Has(ip) {
			count = info.Update.Len() + info.IP.Len()
			return
		}

		info.Update.Add(ip)
		count = info.Update.Len() + info.IP.Len()
	}

	return
}

func save() {
	UserUV.Range(func(k, v interface{}) bool {
		info := v.(Info)
		if !info.Update.IsEmpty() {
			var ips []string
			info.Update.Range(func(key interface{}) bool {
				ips = append(ips, key.(string))
				info.IP.Add(key.(string))
				return true
			})
			info.Update.Clear()
			_ = sdb.UpdateUVIP(k.(string), ips)
		}
		return true
	})
}
