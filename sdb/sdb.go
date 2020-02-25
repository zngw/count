package sdb

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type CountData struct {
	Title string `json:"title"`
	Url   string `json:"url"`
	Time  int    `json:"time"`
}

func Init(file string) (err error) {
	db, err = sql.Open("sqlite3", file)
	return
}

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
	return
}

func AddCount(name, title, url string) (num int) {
	num = GetCount(name, url)
	if num == -1 {
		num = 1
		// 插入数据
		pre := `INSERT INTO %s (title, url, time) values(?,?,?)`
		stmt, err := db.Prepare(fmt.Sprintf(pre, name))
		if stmt == nil || err != nil {
			return
		}

		_, err = stmt.Exec(title, url, num)
		if err != nil {
			return
		}
	} else {
		num++
		// 更新数据
		pre := `update %s set time=? where url=?`
		stmt, err := db.Prepare(fmt.Sprintf(pre, name))
		if stmt == nil || err != nil {
			return
		}
		_, err = stmt.Exec(num, url)
		if err != nil {
			return
		}
	}

	return
}

func GetCount(name string, url string) (num int) {
	num = -1

	// 查询数据
	query := `SELECT time FROM %s where url = "%s"`
	rows, err := db.Query(fmt.Sprintf(query, name, url))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&num)
		return
	}

	return
}

type CR struct {
	Url  string `json:"url"`  // 地址
	Time int    `json:"time"` // 次数
}

func GetCounts(name string, urls []string) (cr []CR) {
	str := ""
	for _, url := range urls {
		str += fmt.Sprintf(`'%s',`, url)
	}
	str = str[:len(str)-1]

	// 查询数据
	query := `SELECT url,time FROM %s where url in(%s)`
	rows, err := db.Query(fmt.Sprintf(query, name, str))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data CR
		err = rows.Scan(&data.Url, &data.Time)
		cr = append(cr, data)
	}

	return
}

func SortByTime(name string, limit int) (lt []CountData) {
	query := `SELECT * FROM %s ORDER BY time DESC  LIMIT %d`
	rows, err := db.Query(fmt.Sprintf(query, name, limit))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var data CountData
		err = rows.Scan(&id, &data.Title, &data.Url, &data.Time)
		lt = append(lt, data)
	}

	return
}
