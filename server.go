// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"filesync/comm"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8088", "http service address")
var root = flag.String("root", "tmp", "localdir to sync")
var token = flag.String("token", "XXXXXXX", "token check")
var uri= flag.String("uri", "echo", "uri")

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	log.Println("new client connecting")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	log.Println("connected")

	var oldFiles []comm.FileInfo

	checkOK := false

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		//log.Printf("recv: %s", message)

		// 解码
		var request comm.Request
		err = json.Unmarshal(message, &request)
		if err != nil {
			panic(err)
		}
		// 检查token
		if request.Cmd == comm.CheckToken {
			clientToken := string(request.Data)
			if *token == clientToken {
				checkOK = true
			}
		}
		if !checkOK {
			log.Println("token no ok")
			return
		}
		if request.Cmd == comm.ReqFiles {
			var remoteFiles []comm.FileInfo
			err = json.Unmarshal(request.Data, &remoteFiles)
			if err != nil {
				panic(err)
			}

			localFiles, err := comm.FilePathWalkDir(*root)
			if err != nil {
				panic(err)
			}
			newfiles, err := comm.Comparedir(localFiles, remoteFiles)
			if err != nil {
				panic(err)
			}

			for _, value := range newfiles {
				log.Println("send to update: ", value)
				req, err := comm.PackSendFile(*root, value)
				if err != nil {
					log.Println("pack file err:", err)
					break
				}
				err = c.WriteMessage(mt, req)
				if err != nil {
					log.Println("write:", err)
					break
				}
			}
		} else if request.Cmd == comm.WriteFile {
			path := filepath.Join(*root, request.Name)
			log.Println("update save to ", path)
			err = comm.SaveFile(path, request.Data)
			if err != nil {
				panic(err)
			}
		} else if request.Cmd == comm.DeleteFile {
			path := filepath.Join(*root, request.Name)
			log.Println("delete file ", path)
			err = os.RemoveAll(path)
			if err != nil {
				panic(err)
			}
		}

		// 校验删除文件
		localFiles, err := comm.FilePathWalkDir(*root)
		if err != nil {
			log.Println("get files :", err)
			return
		}

		delteFiles, err := comm.ToDelete(oldFiles, localFiles)
		if err != nil {
			log.Println("get files :", err)
			return
		}

		for _, name := range delteFiles {
			log.Println("send to delete: ", name)
			req, err := comm.PackDeleteFile(name)
			if err != nil {
				panic(err)
			}
			err = c.WriteMessage(mt, req)
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
		oldFiles = localFiles // 更新列表

		req, err := comm.PackSendFileList(*root)
		if err != nil {
			log.Println("get file list", err)
			break
		}
		err = c.WriteMessage(mt, req)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/"+*uri, echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
