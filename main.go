package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zngw/count/sdb"
	"github.com/zngw/count/uv"
	"github.com/zngw/log"
	"io/ioutil"
	"net"
	"net/http"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

// 配置文件结构体
type Config struct {
	LogFile string   `json:"log"`    // DB文件
	LogTag  []string `json:"logTag"` // 日志输出类型
	Addr    string   `json:"addr"`   // 端口
	DB      string   `json:"db"`     // DB文件
	User    []string `json:"user"`   // 启用用户名
}

func (p *Config) CheckUser(user string) bool {
	for i, _ := range p.User {
		if user == p.User[i] {
			return true
		}
	}

	return false
}

var Cfg Config

func main() {
	cfg := flag.String("c", "./config.json", "默认配置为 ./config.json")
	flag.Parse()

	// 读取配置
	fmt.Println("读取配置文件:", *cfg)
	raw, err := ioutil.ReadFile(*cfg)
	if err != nil {
		panic(err)
		return
	}

	// 序列化配置数据
	err = json.Unmarshal(raw, &Cfg)
	if err != nil {
		log.Error(err)
		return
	}

	// 初始始日志
	fmt.Println("日志路径:", Cfg.LogFile)
	err = log.Init(Cfg.LogFile, Cfg.LogTag)
	if err != nil {
		panic(err)
		return
	}

	// 初始化数据库
	log.Trace("sys", "初始化数据库")
	err = sdb.Init(Cfg.DB)
	if err != nil {
		log.Error(err)
		return
	}

	// 初始化UV信息
	uv.Init(Cfg.User)

	// 根据用户创建表，一个用户一个表
	for _, user := range Cfg.User {
		err = sdb.CreateTable(user)
		if err != nil {
			log.Error(err)
			return
		}
	}

	// 监听事件
	http.HandleFunc("/count/add", add) // 增加次数
	http.HandleFunc("/count/get", get) // 获取次数
	http.HandleFunc("/count/top", top) // 获取排行
	go func() {
		err = http.ListenAndServe(Cfg.Addr, nil) // 设置监听的端口
		if err != nil {
			log.Error(err)
		}
	}()

	log.Trace("sys", "服务器启动成功：", Cfg.Addr)
	signal.Ignore(syscall.SIGHUP)
	runtime.Goexit()
}

// clientIP 尽最大努力实现获取客户端 IP 的算法。
// 解析 X-Real-IP 和 X-Forwarded-For 以便于反向代理（nginx 或 haproxy）可以正常工作。
func clientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if ip != "" {
		return ip
	}

	ip = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return ""
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
		Host  string `json:"host"`  // 来源网址
	}

	var data tmp
	err := json.Unmarshal(param, &data)
	if err != nil {
		send(w, []byte(err.Error()))
		return
	}

	if !Cfg.CheckUser(data.User) {
		send(w, []byte(`{"time":0}`))
	}

	ip := clientIP(r)
	var num = 0
	log.Trace("record", ip, "->", data.Host+data.Url, "[", data.Title, "]")
	// 排除localhost统计
	if strings.Index(data.Host, "localhost") == -1 {
		num = sdb.AddCount(data.User, data.Title, data.Url)
	} else {
		num = sdb.GetCount(data.User, data.Url)
	}

	uv := uv.Add(data.User, ip)
	send(w, []byte(fmt.Sprintf(`{"time":%v,"uv":%v}`, num, uv)))
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
		send(w, []byte(`[]`))
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
		send(w, []byte(`[]`))
	}

	ls := sdb.SortByTime(data.User, data.Limit)
	if ls == nil {
		send(w, []byte("[]"))
		return
	}

	b, err := json.Marshal(ls)
	send(w, b)
}
