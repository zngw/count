// @Title
// @Description $
// @Author  55
// @Date  2021/3/5
package data

import "sync"

var DataMap sync.Map //根据数据特性这里用降序list

type CountData struct {
	Title  string `json:"title"  bson:"title,omitempty"`
	Url    string `json:"url"    bson:"url,omitempty"`
	Time   int    `json:"time"   bson:"time,omitempty"`
	Update bool   `json:"update" bson:"update,omitempty"`
	User   string `json:"user"   bson:"user,omitempty"`
	Ip     string `json:"ip"     bson:"ip,omitempty"`
	Date   int64  `json:"date"   bson:"date,omitempty"`
}

type CR struct {
	Url  string `json:"url"  json:"url"`  // 地址
	Time int    `json:"time" json:"time"` // 次数
}

type UV struct {
	Ip string `json:"ip"  json:"ip"` // 地址
}
