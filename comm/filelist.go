package comm

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileInfo struct {
	Path  string    `json:"path"`
	Dtime time.Time `json:"dtime"`
	Md5   []byte    `json:"md5"`
}

var oldFiles = struct {
	sync.RWMutex
	m map[string]FileInfo
}{m: make(map[string]FileInfo)}

// 支持缓存读取
func getFileMd5(path string) ([]byte, error) {
	var sum []byte
	fileInfo, err := os.Stat(path)
	if err != nil {
		return sum, err
	}
	modTime := fileInfo.ModTime()
	oldFiles.RLock()
	v, ok := oldFiles.m[path]
	oldFiles.RUnlock()
	if ok {
		if v.Dtime.Equal(modTime) {
			sum = v.Md5
			return sum, nil
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return sum, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return sum, err
	}

	sum = h.Sum(nil)
	oldFiles.Lock()
	oldFiles.m[path] = FileInfo{Path: path, Dtime: modTime, Md5: sum}
	oldFiles.Unlock()
	return sum, nil
}

func FilePathWalkDir(root string) ([]FileInfo, error) {
	var files []FileInfo
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".svn") {
			return nil
		}
		if strings.Contains(path, ".git") {
			return nil
		}
		if !info.IsDir() {
			fileMd5, err := getFileMd5(path)
			if err != nil {
				log.Fatal(err)
			}
			fullPath := strings.TrimPrefix(path, root)
			normalPath := filepath.ToSlash(fullPath)
			files = append(files, FileInfo{Path: normalPath, Dtime: info.ModTime(), Md5: fileMd5})
		}
		return nil
	})
	return files, err
}

// src需要更新到dst的文件：
// src有，dst没有
// src比dts更新,且有修改
func Comparedir(src []FileInfo, dst []FileInfo) ([]string, error) {
	var paths []string
	var find FileInfo
	for _, s1 := range src {
		found := false

		for _, d1 := range dst {
			if s1.Path == d1.Path {
				found = true
				find = d1
				break
			}
		}
		if !found {
			paths = append(paths, s1.Path)
		} else {
			if s1.Dtime.After(find.Dtime) {
				if !bytes.Equal(s1.Md5, find.Md5) {
					paths = append(paths, s1.Path)
				}
			}
		}

	}

	return paths, nil
}

// 需要删除的文件
// old 有 new 没有
func ToDelete(old []FileInfo, new []FileInfo) ([]string, error) {
	var paths []string
	for _, s1 := range old {
		found := false
		for _, d1 := range new {
			if s1.Path == d1.Path {
				found = true
				break
			}
		}
		if !found {
			paths = append(paths, s1.Path)
		}
	}

	return paths, nil
}

func packFileInfo(path string) ([]byte, error) {
	var data []byte
	files, err := FilePathWalkDir(path)
	if err != nil {
		panic(err)
	}
	// 编码
	data, err = json.Marshal(files)
	if err != nil {
		log.Println("error:", err)
	}
	return data, nil
}

func SaveFile(path string, data []byte) error {
	filedir := filepath.Dir(path)
	err := os.MkdirAll(filedir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// FIXME: root value 区分
func PackSendFile(root string, value string) ([]byte, error) {
	var data []byte
	path := filepath.Join(root, value)
	p, err := ioutil.ReadFile(path)
	if err != nil {
		return data, err
	}
	request := Request{
		Cmd:  WriteFile,
		Name: value,
		Data: p,
	}

	data, err = json.Marshal(request)
	if err != nil {
		return data, err
	}
	return data, nil
}

func PackDeleteFile(name string) ([]byte, error) {
	var data []byte
	request := Request{
		Cmd:  DeleteFile,
		Name: name,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return data, err
	}
	return data, nil
}

func PackSendFileList(root string) ([]byte, error) {
	var data []byte
	msg, err := packFileInfo(root)
	if err != nil {
		return data, err
	}

	request := Request{
		Cmd:  ReqFiles,
		Data: msg,
	}

	data, err = json.Marshal(request)
	if err != nil {
		return data, err
	}
	return data, nil
}
