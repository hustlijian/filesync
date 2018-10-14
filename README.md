# filesync
使用http同步文件

## 使用
* 安装依赖
```
go get github.com/gorilla/websocket
```

* 下载
下载文件到 $GOPATH/src 目录下

* 编译
```
cd filesync && make
```

* 使用
服务器(linux/windows)：
```
Usage of ./server:
  -addr string
    	http service address (default "localhost:8088")
  -root string
    	localdir to sync (default "tmp")
  -token string
          token check (default "XXXXXXX")

```

客户端(windows/linux)：
```
Usage of client.exe:
  -addr string
        http service address (default "localhost:8088")
  -cycle int
        cycle to check (default 3)
  -encrpt
        encrypt or not
  -root string
        localdir to sync (default "b")
  -token string
        token check (default "XXXXXXX")
```

## TODO

- [X] windows,linux客户端兼容
因为linux下创建目录权限问题，需要区分做编译支持
- [X] md5缓存，只有时间变更时才更新
- [X] 加入token校验
- [ ] link 链接文件处理
- [ ] 文件过滤配置

## 备注
- 安装golang.org/x/ 下库
```
mkdir -p $GOPATH/src/golang.org/x/
cd !$
git clone https://github.com/golang/net.git
git clone https://github.com/golang/sys.git
git clone https://github.com/golang/tools.git
```