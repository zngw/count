package sdb

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zngw/set"
	"sort"
	"sync"
	"time"
)

type CountData struct {
	Title string `json:"title"`
	Url   string `json:"url"`
	Time  int    `json:"time"`
	Update bool  `json:"update"`
}

var db *sql.DB
var dataMap sync.Map //根据数据特性这里用降序list
var lock sync.Mutex //记数互斥锁

func Init(file string) (err error) {
	db, err = sql.Open("sqlite3", file)

	go func() {
		for true {
			save()
			time.Sleep(time.Second)
		}
	}()
	return
}

// 文章浏览次数 ===========================================
func CreateTable(name string) (err error) {
	table := `
    CREATE TABLE IF NOT EXISTS %s (
        uid INTEGER PRIMARY KEY AUTOINCREMENT,
        title VARCHAR(128) NULL,
        url VARCHAR(64) NULL,
		time INTEGER NULL
    );
    `
	cmd := fmt.Sprintf(table, name)
	_, err = db.Exec(cmd)
	if err != nil {
		return
	}

	// 读取所有数据到内存
	query := `SELECT title, url, time FROM %s`
	rows, err := db.Query(fmt.Sprintf(query, name))
	if err != nil {
		return
	}
	defer rows.Close()

	dataList := make([]*CountData,0)
	for rows.Next() {
		data := new(CountData)
		err = rows.Scan(&data.Title,&data.Url,&data.Time)
		if err != nil {
			return
		}

		dataList = append(dataList,data)
	}

	// 阅读次数Time降顺
	sort.Slice(dataList, func(i, j int) bool {
		return dataList[i].Time > dataList[j].Time
	})

	dataMap.Store(name,&dataList)
	return
}

func save()  {
	dataMap.Range(func(k, v interface{}) bool {
		dataList := v.(*[]*CountData)
		//dataList := v.([]*CountData)
		for i,_ := range *dataList {
			data := (*dataList)[i]
			if data.Update {
				data.Update = false

				// 更新数据
				pre := `update %s set time=? where url=?`
				stmt, err := db.Prepare(fmt.Sprintf(pre, k))
				if stmt == nil || err != nil {
					return true
				}
				// data是批针指向dataMap中的数据，如多线程顺序乱了也不会影响最终写的数据为内存中的值
				_, _ = stmt.Exec(data.Time, data.Url)
				_ = stmt.Close()
			}
		}

		return true
	})
}

func AddCount(name, title, url string) int {
	v, ok := dataMap.Load(name)
	if !ok {
		return 0
	}
	//dataList := v.([]*CountData)
	dataList := v.(*[]*CountData)
	lock.Lock()
	defer lock.Unlock()
	for i,_ := range *dataList{
		data := (*dataList)[i]
		if data.Url == url {
			// 存在
			data.Time++
			data.Update = true

			// 与前一个比较，是否需要调整顺序
			if i > 0 && data.Time>(*dataList)[i-1].Time {
				(*dataList)[i] = (*dataList)[i-1]
				(*dataList)[i-1] = data
			}
			return data.Time
		}
	}

	// 不存在，插入
	data := new(CountData)
	data.Url = url
	data.Time = 1
	data.Title = title
	*dataList = append(*dataList, data)

	go func() {
		// 插入数据
		pre := `INSERT INTO %s (title, url, time) values(?,?,?)`
		stmt, err := db.Prepare(fmt.Sprintf(pre, name))
		if stmt == nil || err != nil {
			return
		}

		_, _ = stmt.Exec(data.Title, data.Url, data.Time)
		_ = stmt.Close()
	}()

	return data.Time
}

func GetCount(name string, url string)  int {
	v, ok := dataMap.Load(name)
	if !ok {
		return 0
	}
	dataList := v.(*[]*CountData)

	for i,_ := range *dataList {
		data := (*dataList)[i]
		if data.Url == url {
			return data.Time
		}
	}

	return 0
}

type CR struct {
	Url  string `json:"url"`  // 地址
	Time int    `json:"time"` // 次数
}

func GetCounts(name string, urls []string) (cr []CR) {
	v, ok := dataMap.Load(name)
	if !ok {
		return
	}
	dataList := v.(*[]*CountData)

	for i,_ := range *dataList {
		for j, _ := range urls{
			if (*dataList)[i].Url == urls[j] {
				var data CR
				data.Url = (*dataList)[i].Url
				data.Time = (*dataList)[i].Time
				cr = append(cr, data)
				break
			}
		}
	}

	return
}

func SortByTime(name string, limit int) (lt []CountData) {
	v, ok := dataMap.Load(name)
	if !ok {
		return
	}
	dataList := v.(*[]*CountData)

	for i,_ := range *dataList {
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
	table := `
    CREATE TABLE IF NOT EXISTS uv_%s (
        uid INTEGER PRIMARY KEY AUTOINCREMENT,
        ip VARCHAR(16) NULL
    );
    `
	cmd := fmt.Sprintf(table, name)
	_, err = db.Exec(cmd)
	return
}

func GetUVIPList(name string) (list *set.Set) {
	query := `SELECT ip FROM uv_%s`
	rows, err := db.Query(fmt.Sprintf(query, name))
	if err != nil {
		return
	}
	defer rows.Close()

	list = set.New()
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		list.Add(ip)
	}
	return
}

func UpdateUVIP(name string, ips []string) (err error) {
	pre := `INSERT INTO uv_%s (ip) values(?)`
	stmt, err := db.Prepare(fmt.Sprintf(pre, name))
	if stmt == nil || err != nil {
		return
	}

	defer stmt.Close()

	for _, ip := range ips{
		_, err = stmt.Exec(ip)
		if err != nil {
			return
		}
	}

	return
}
// UV数据结束
