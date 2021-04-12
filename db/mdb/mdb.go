// @Title
// @Description $
// @Author  55
// @Date  2021/3/6
package mdb

import (
	"fmt"
	"github.com/zngw/count/cfg"
	"github.com/zngw/count/data"
	"github.com/zngw/log"
	"github.com/zngw/set"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sync"
	"time"
)

const DbcLog = "Log"     // log
const DbcCount = "Count" // 记数
const DbcUV = "UV"       // UV

var dbs map[string]*mgo.Database
var lock sync.Mutex //记数互斥锁

func Init() (err error) {
	dbs = make(map[string]*mgo.Database)
	for _, name := range cfg.Cfg.User {
		s, err := mgo.Dial(cfg.Cfg.DBUrl)
		if err != nil {
			return err
		}

		dbs[name] = s.DB(name)
	}

	return
}

func dbc(user, name string) *mgo.Collection {
	db, ok := dbs[user]
	if !ok {
		return nil
	}

	return db.C(name)
}

// 文章浏览次数 ===========================================
func CreateTable(name string) (err error) {

	// 读取所有数据到内存
	c := dbc(name, DbcCount)
	var dataList []data.CountData
	err = c.Find(nil).All(&dataList)
	if err != nil {
		err = fmt.Errorf("加载失败分类失败：%v", err)
		return
	}

	dl := make([]*data.CountData, 0)
	for _, d := range dataList {
		dt := new(data.CountData)
		dt.Update = false
		dt.Title = d.Title
		dt.Time = d.Time
		dt.Url = d.Url
		dt.User = d.User
		dl = append(dl, dt)
	}

	data.DataMap.Store(name, &dl)
	return
}

func Save() {
	data.DataMap.Range(func(k, v interface{}) bool {
		dataList := v.(*[]*data.CountData)
		//dataList := v.([]*CountData)
		for i, _ := range *dataList {
			dt := (*dataList)[i]
			if dt.Update {
				c := dbc(dt.User, DbcCount)
				err := c.Update(bson.M{"url": dt.Url}, bson.M{"$set": bson.M{"time": dt.Time}})
				if err != nil {
					if err.Error() == "not found" {
						err = c.Insert(dt)
						if err != nil {
							log.Error(err)
						}
					} else {
						log.Error(err)
					}
				}
				dt.Update = false
			}
		}

		return true
	})
}

func AddCount(name, title, url, ip string) int {
	v, ok := data.DataMap.Load(name)
	if !ok {
		return 0
	}

	//dataList := v.([]*CountData)
	dataList := v.(*[]*data.CountData)
	lock.Lock()
	defer lock.Unlock()

	var dt *data.CountData = nil
	for i, _ := range *dataList {
		tmp := (*dataList)[i]
		if tmp.Url == url {
			// 存在
			dt = tmp
			dt.Time++
			dt.Update = true

			// 与前一个比较，是否需要调整顺序
			if i > 0 && dt.Time > (*dataList)[i-1].Time {
				(*dataList)[i] = (*dataList)[i-1]
				(*dataList)[i-1] = dt
			}
			break
		}
	}

	if dt == nil {
		// 不存在，插入
		dt = new(data.CountData)
		dt.User = name
		dt.Url = url
		dt.Time = 1
		dt.Title = title
		*dataList = append(*dataList, dt)

		go func() {
			// 插入数据
			c := dbc(name, DbcCount)
			err := c.Insert(dt)
			if err != nil {
				return
			}
		}()
	}

	dt.Date = time.Now().Unix()
	dt.Ip = ip

	// 插入流水日志
	go func() {
		// 插入流水数据
		c := dbc(name, DbcLog)
		err := c.Insert(dt)
		if err != nil {
			return
		}
	}()

	return dt.Time
}

func GetCount(name string, url string) int {
	v, ok := data.DataMap.Load(name)
	if !ok {
		return 0
	}
	dataList := v.(*[]*data.CountData)

	for i, _ := range *dataList {
		dt := (*dataList)[i]
		if dt.Url == url {
			return dt.Time
		}
	}

	return 0
}

func GetCounts(name string, urls []string) (cr []data.CR) {
	v, ok := data.DataMap.Load(name)
	if !ok {
		return
	}
	dataList := v.(*[]*data.CountData)

	for i, _ := range *dataList {
		for j, _ := range urls {
			if (*dataList)[i].Url == urls[j] {
				var dt data.CR
				dt.Url = (*dataList)[i].Url
				dt.Time = (*dataList)[i].Time
				cr = append(cr, dt)
				break
			}
		}
	}

	return
}

func SortByTime(name string, limit, typ int) (lt []data.CountData) {
	now := time.Now()
	var begin int64 = 0

	year := now.Year()
	month := now.Month()
	day := now.Day()

	if typ == 1 {
		// 当天
		begin = time.Date(year, month, day, 0, 0, 0, 0, time.Local).Unix()
	} else if typ == 2 {
		begin = time.Date(year, month, day, 0, 0, 0, 0, time.Local).Unix()
		week := now.Weekday()
		begin -= (int64)(week * 86400)
	} else if typ == 3 {
		// 当月
		begin = time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Unix()
	} else if typ == 4 {
		// 当年
		begin = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local).Unix()
	}

	c := dbc(name, DbcLog)
	m := []bson.M{
		{"$match": bson.M{"user": name, "url": bson.M{"$ne": "/"}, "date": bson.M{"$gte": begin}}},
		{"$group": bson.M{"_id": "$url", "time": bson.M{"$sum": 1}, "title": bson.M{"$first": "$title"}, "url": bson.M{"$first": "$url"}}},
		{"$sort": bson.M{"time": -1}},
		{"$limit": limit},
	}

	err := c.Pipe(m).All(&lt)
	if err != nil {
		return
	}
	return
}

// 文章浏览次数结束 ===============================================

// UV数据 ========================================================
func CreateUVTable(name string) (err error) {
	return
}

func GetUVIPList(name string) (list *set.Set) {
	list = set.New()

	c := dbc(name, DbcUV)
	var dataList []data.UV
	err := c.Find(nil).All(&dataList)
	if err != nil {
		return
	}

	for _, uv := range dataList {
		list.Add(uv.Ip)
	}
	return
}

func UpdateUVIP(name string, ips []string) (err error) {
	c := dbc(name, DbcUV)
	for _, ip := range ips {
		err = c.Insert(bson.M{"ip": ip})
	}
	return
}

// UV数据结束
