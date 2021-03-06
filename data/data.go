// @Title
// @Description $
// @Author  55
// @Date  2021/3/5
package data

import "sync"

var DataMap sync.Map //根据数据特性这里用降序list

type CountData struct {
	Title  string `json:"title"  bson:"title"`
	Url    string `json:"url"    bson:"url"`
	Time   int    `json:"time"   bson:"time"`
	Update bool   `json:"update" bson:"update"`
	User   string `json:"user"   bson:"user"`
}

type CR struct {
	Url  string `json:"url"  json:"url"`  // 地址
	Time int    `json:"time" json:"time"` // 次数
}

type UV struct {
	Ip string `json:"ip"  json:"ip"` // 地址
}
