// @Title
// @Description $
// @Author  55
// @Date  2021/3/5
package cfg

// 配置文件结构体
type Config struct {
	LogFile string   `json:"log"`    // DB文件
	LogTag  []string `json:"logTag"` // 日志输出类型
	Addr    string   `json:"addr"`   // 端口
	DBType  string   `json:"dbType"` // DB类型 mdb-mongodb,sdb-sqlite3
	DBFile  string   `json:"dbFile"` // DB文件
	DBUrl   string   `json:"dbUrl"`  // DB配置
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
