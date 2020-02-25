package main

import (
	"encoding/json"
	"fmt"
	"github.com/zngw/count/sdb"
	"io/ioutil"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"
)

// 配置文件结构体
type Config struct {
	Addr string   `json:"addr"` // 端口
	DB   string   `json:"db"`   // DB文件
	User []string `json:"user"` // 启用用户名
}

func (p *Config)CheckUser(user string) bool {
	for i,_ := range p.User{
		if user == p.User[i] {
			return true
		}
	}

	return false
}

var Cfg Config

func main() {
	// 读取配置
	raw, err := ioutil.ReadFile("./config.json")
	if err != nil {
		panic(err)
		return
	}

	// 序列化配置数据
	err = json.Unmarshal(raw, &Cfg)
	if err != nil {
		panic(err)
		return
	}

	// 初始化数据库
	err = sdb.Init(Cfg.DB)
	if err != nil {
		panic(err)
		return
	}

	//根据用户创建表，一个用户一个表
	for _, user := range Cfg.User {
		err = sdb.CreateTable(user)
		if err != nil {
			panic(err)
			return
		}
	}

	// 监听事件
	http.HandleFunc("/count/add", add)       // 增加次数
	http.HandleFunc("/count/get", get)       // 获取次数
	http.HandleFunc("/count/top", top)       // 获取排行
	err = http.ListenAndServe(Cfg.Addr, nil) // 设置监听的端口
	if err != nil {
		panic(err)
	}

	signal.Ignore(syscall.SIGHUP)
	runtime.Goexit()
}

// 封装发送接口
func send(w http.ResponseWriter, data []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")             //返回数据格式是数据流
	_, err := w.Write(data)
	if err != nil {
		fmt.Println(err)
	}
}

// 增加次数
func add(w http.ResponseWriter, r *http.Request) {
	param, _ := ioutil.ReadAll(r.Body)
	_ = r.Body.Close()

	type tmp struct {
		User  string `json:"user"`  // 用户
		Title string `json:"title"` // 标题
		Url   string `json:"url"`   // 地址
	}

	var data tmp
	err := json.Unmarshal(param, &data)
	if err != nil {
		send(w, []byte(err.Error()))
		return
	}

	if !Cfg.CheckUser(data.User) {
		send(w,[]byte(`{"time":0}`))
	}

	num := sdb.AddCount(data.User, data.Title, data.Url)
	send(w, []byte(fmt.Sprintf(`{"time":%v}`, num)))
}

// 获取次数
func get(w http.ResponseWriter, r *http.Request) {
	param, _ := ioutil.ReadAll(r.Body)
	_ = r.Body.Close()

	type tmp struct {
		User string   `json:"user"` // 用户
		Url  []string `json:"url"`  // 地址
	}

	var data tmp
	err := json.Unmarshal(param, &data)
	if err != nil {
		send(w, []byte(err.Error()))
		return
	}

	if !Cfg.CheckUser(data.User) {
		send(w,[]byte(`[]`))
	}

	results := sdb.GetCounts(data.User, data.Url)
	b, err := json.Marshal(results)
	send(w, b)
}

// 获取排行数据
func top(w http.ResponseWriter, r *http.Request) {
	param, _ := ioutil.ReadAll(r.Body)
	_ = r.Body.Close()

	type tmp struct {
		User  string `json:"user"`  // 用户
		Limit int    `json:"limit"` // 限制数
	}

	var data tmp
	err := json.Unmarshal(param, &data)
	if err != nil {
		send(w, []byte(err.Error()))
		return
	}

	if !Cfg.CheckUser(data.User) {
		send(w,[]byte(`[]`))
	}

	ls := sdb.SortByTime(data.User, data.Limit)
	if ls == nil {
		send(w, []byte("[]"))
		return
	}

	b, err := json.Marshal(ls)
	send(w, b)
}
