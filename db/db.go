package db

import (
	"github.com/zngw/count/cfg"
	"github.com/zngw/count/data"
	"github.com/zngw/count/db/mdb"
	"github.com/zngw/count/db/sdb"
	"github.com/zngw/set"
	"time"
)

func Init() (err error) {
	if cfg.Cfg.DBType == "mdb" {
		err = mdb.Init()
	} else if cfg.Cfg.DBType == "sdb" {
		err = sdb.Init()
	}

	go func() {
		for true {
			Save()
			time.Sleep(time.Second)
		}
	}()

	return
}

func Save() {
	if cfg.Cfg.DBType == "mdb" {
		mdb.Save()
	} else if cfg.Cfg.DBType == "sdb" {
		sdb.Save()
	}
}

// 文章浏览次数 ===========================================
func CreateTable(name string) (err error) {
	if cfg.Cfg.DBType == "mdb" {
		err = mdb.CreateTable(name)
	} else if cfg.Cfg.DBType == "sdb" {
		err = sdb.CreateTable(name)
	}
	return
}

func AddCount(name, title, url string) int {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.AddCount(name, title, url)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.AddCount(name, title, url)
	}

	return 0
}

func GetCount(name, url string) int {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.GetCount(name, url)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.GetCount(name, url)
	}

	return 0
}

func GetCounts(name string, urls []string) (cr []data.CR) {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.GetCounts(name, urls)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.GetCounts(name, urls)
	}

	return
}

func SortByTime(name string, limit int) (lt []data.CountData) {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.SortByTime(name, limit)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.SortByTime(name, limit)
	}

	return
}

// 文章浏览次数结束 ===============================================

// UV数据 ========================================================
func CreateUVTable(name string) (err error) {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.CreateUVTable(name)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.CreateUVTable(name)
	}

	return
}

func GetUVIPList(name string) (list *set.Set) {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.GetUVIPList(name)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.GetUVIPList(name)
	}
	return
}

func UpdateUVIP(name string, ips []string) (err error) {
	if cfg.Cfg.DBType == "mdb" {
		return mdb.UpdateUVIP(name, ips)
	} else if cfg.Cfg.DBType == "sdb" {
		return sdb.UpdateUVIP(name, ips)
	}

	return
}

// UV数据结束
