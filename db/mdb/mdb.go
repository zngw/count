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
)

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

	lst := make([]*data.CountData, 0)
	for _, d := range dataList {
		lst = append(lst, &d)
	}

	data.DataMap.Store(name, &lst)
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

func AddCount(name, title, url string) int {
	v, ok := data.DataMap.Load(name)
	if !ok {
		return 0
	}
	//dataList := v.([]*CountData)
	dataList := v.(*[]*data.CountData)
	lock.Lock()
	defer lock.Unlock()
	for i, _ := range *dataList {
		dt := (*dataList)[i]
		if dt.Url == url {
			// 存在
			dt.Time++
			dt.Update = true

			// 与前一个比较，是否需要调整顺序
			if i > 0 && dt.Time > (*dataList)[i-1].Time {
				(*dataList)[i] = (*dataList)[i-1]
				(*dataList)[i-1] = dt
			}
			return dt.Time
		}
	}

	// 不存在，插入
	dt := new(data.CountData)
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

func SortByTime(name string, limit int) (lt []data.CountData) {
	v, ok := data.DataMap.Load(name)
	if !ok {
		return
	}
	dataList := v.(*[]*data.CountData)

	for i, _ := range *dataList {
		if i >= limit {
			break
		}

		lt = append(lt, *(*dataList)[i])
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
