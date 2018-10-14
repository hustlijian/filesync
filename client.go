// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"filesync/comm"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8088", "http service address")
var encrypt = flag.Bool("encrpt", false, "encrypt or not")
var root = flag.String("root", "b", "localdir to sync")
var token = flag.String("token", "XXXXXXX", "token check")
var cycle = flag.Int("cycle", 3, "cycle to check")
var uri= flag.String("uri", "echo", "uri")

func main() {
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	scheme := "ws"
	if *encrypt {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: *addr, Path: "/"+*uri}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	log.Printf("connected")

	done := make(chan struct{})
	var writeMutex sync.Mutex
	syncFlag := false // 是否可以同步
	var oldFiles []comm.FileInfo

	// token 校验
	req, err := comm.PackCheckToken(*token)
	if err != nil {
		log.Fatal("pack token ", err)
	}
	writeMutex.Lock()
	err = c.WriteMessage(websocket.TextMessage, req)
	writeMutex.Unlock()
	if err != nil {
		log.Fatal("write:", err)
	}
	syncFlag = true

	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			// log.Printf("recv: %s", message)

			// 解码
			var request comm.Request
			err = json.Unmarshal(message, &request)
			if err != nil {
				panic(err)
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
						panic(err)
					}
					writeMutex.Lock()
					err = c.WriteMessage(mt, req)
					writeMutex.Unlock()
					if err != nil {
						log.Println("write:", err)
						break
					}
				}
				syncFlag = true // 接收完对面的列表后，才可以发自己的列表
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
		}
	}()

	seconds := time.Duration(*cycle) * time.Second
	ticker := time.NewTicker(seconds)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			fmt.Printf("\rtry update: %s", t.Format("15:04:05"))
			if !syncFlag {
				log.Println("syncing ...")
				continue
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
				writeMutex.Lock()
				err = c.WriteMessage(websocket.TextMessage, req)
				writeMutex.Unlock()
				if err != nil {
					log.Println("write:", err)
					break
				}
			}
			oldFiles = localFiles // 更新列表

			req, err := comm.PackSendFileList(*root)
			if err != nil {
				log.Println("get file list", err)
				return
			}
			writeMutex.Lock()
			err = c.WriteMessage(websocket.TextMessage, req)
			writeMutex.Unlock()
			if err != nil {
				log.Println("write:", err)
				return
			}

			syncFlag = false
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
