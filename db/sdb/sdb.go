package sdb

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zngw/count/cfg"
	"github.com/zngw/count/data"
	"github.com/zngw/set"
	"sort"
	"sync"
	"time"
)

var db *sql.DB
var lock sync.Mutex //记数互斥锁

func Init() (err error) {
	db, err = sql.Open("sqlite3", cfg.Cfg.DBFile)

	return
}

// 文章浏览次数 ===========================================
func CreateTable(name string) (err error) {
	table := `
    CREATE TABLE IF NOT EXISTS log_%s (
        uid INTEGER PRIMARY KEY AUTOINCREMENT,
        title VARCHAR(128) NULL,
        url VARCHAR(64) NULL,
		time INTEGER NULL,
		ip VARCHAR(32) NULL,
		date INTEGER NULL
    );
    `
	cmd := fmt.Sprintf(table, name)
	_, err = db.Exec(cmd)
	if err != nil {
		return
	}

	table = `
    CREATE TABLE IF NOT EXISTS %s (
        uid INTEGER PRIMARY KEY AUTOINCREMENT,
        title VARCHAR(128) NULL,
        url VARCHAR(64) NULL,
		time INTEGER NULL
    );
    `
	cmd = fmt.Sprintf(table, name)
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

	dataList := make([]*data.CountData, 0)
	for rows.Next() {
		dt := new(data.CountData)
		err = rows.Scan(&dt.Title, &dt.Url, &dt.Time)
		if err != nil {
			return
		}

		dataList = append(dataList, dt)
	}

	// 阅读次数Time降顺
	sort.Slice(dataList, func(i, j int) bool {
		return dataList[i].Time > dataList[j].Time
	})

	data.DataMap.Store(name, &dataList)
	return
}

func Save() {
	data.DataMap.Range(func(k, v interface{}) bool {
		dataList := v.(*[]*data.CountData)
		//dataList := v.([]*CountData)
		for i, _ := range *dataList {
			dt := (*dataList)[i]
			if dt.Update {
				dt.Update = false

				// 更新数据
				pre := `update %s set time=? where url=?`
				stmt, err := db.Prepare(fmt.Sprintf(pre, k))
				if stmt == nil || err != nil {
					return true
				}
				// data是批针指向dataMap中的数据，如多线程顺序乱了也不会影响最终写的数据为内存中的值
				_, _ = stmt.Exec(dt.Time, dt.Url)
				_ = stmt.Close()
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
		dt.Url = url
		dt.Time = 1
		dt.Title = title
		*dataList = append(*dataList, dt)

		go func() {
			// 插入数据
			pre := `INSERT INTO %s (title, url, time) values(?,?,?)`
			stmt, err := db.Prepare(fmt.Sprintf(pre, name))
			if stmt == nil || err != nil {
				return
			}

			_, _ = stmt.Exec(dt.Title, dt.Url, dt.Time)
			_ = stmt.Close()
		}()
	}

	dt.Date = time.Now().Unix()
	dt.Ip = ip

	go func() {
		// 插入数据
		pre := `INSERT INTO log_%s (title, url, time, ip, date) values(?,?,?,?,?)`
		stmt, err := db.Prepare(fmt.Sprintf(pre, name))
		if stmt == nil || err != nil {
			return
		}

		_, _ = stmt.Exec(dt.Title, dt.Url, dt.Time, dt.Ip, dt.Date)
		_ = stmt.Close()
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

	query := `SELECT SUM(1), title, url FROM log_%s WHERE url != "/" AND url != "" GROUP BY url ORDER BY SUM(1) DESC Limit %d`
	rows, err := db.Query(fmt.Sprintf(query, name, limit))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var num int64
		var title, url string
		err = rows.Scan(&num, &title, &url)
		if err != nil {
			continue
		}

		var cd data.CountData
		cd.Url = url
		cd.Title = title
		cd.Time = int(num)
		cd.User = name
		lt = append(lt, cd)
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
	list = set.New()

	query := `SELECT ip FROM uv_%s`
	rows, err := db.Query(fmt.Sprintf(query, name))
	if err != nil {
		return
	}
	defer rows.Close()

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

	for _, ip := range ips {
		_, err = stmt.Exec(ip)
		if err != nil {
			return
		}
	}

	return
}

// UV数据结束
