# 一个基于SQLite数据库的Web计数服务器
## 1. 前提  
最近用hexo写了个静态[博客](https://zengwu.com.cn)，在文章阅读次数的时候用LeanCloud记数时出现了429太多请求的错误，于是自己写了一个简单的计数器

## 2. 使用
2.1 去发布下载对应系统的版本文件，或下载源码自行编译。  
2.2 修改配置文件`config.json`中的地址端口和数据库的名字。
```json
{
  "addr":":80",
  "db":"count.db",
  "user":[ "zngw"]
}
``` 
* "addr": web服务器监听的ip端口,`:`前为空则监听任意IP
* "db": SQLite数据库文件
* "user": 用户数组，只有存在的用户可以记录值
2.3 运行下载或编译好的`count`程序

## 3. 在Hexo中的使用
在Hexo next主题中使用详见文章[https://zengwu.com.cn/p/7b5001f2.html](https://zengwu.com.cn/p/7b5001f2.html)

# 接口说明
通过post收发json格式数据
1. add 增加  
post： '/count/add'  
发送数据
```json
{
  "user":"zngw", // 用户，只有配置文件里的用户可记数
  "title":"文件标题",  // 标题
  "url": "跳转地址"    // 地址
}
```
返回数据
```json
{
  "time": 1,     // url地址访问后的次数
  "uv":          // ip过滤访问次数
}
```

2. get 获取次数
post: '/count/get'
发送数据
```json
{
  "user":"guoke3915",           // 用户
  "url": ["/","/p/abcdef.html"] // 查询的地址
}
```
返回数据
```json
[
  {
    "url": "/",  // 查询地址
    "time": 10   // 访问次数
  },
  {
    "url": "/p/abcdef.html",  // 查询地址
    "time": 1                 // 访问次数
  }
]
```

3. top 访问排行
post: '/count/top'
发送数据
```json
{
  "user": "guoke3915",    // 用户
  "limit": 24,            // 查询数量
}
```
返回数据
```json
[
  {
    "title": "标题",
    "url": "/",     // 查询地址
    "time": 29      // 访问数据
  },
  {
    "title": "标题",
    "url": "/p/abcdef.html", // 查询地址
    "time": 9                // 访问数据
  }
]
```