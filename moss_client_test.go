// @Date: 2021/1/11 21:17
// @Description:
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMossClient(t *testing.T) {
	fileList := make([]string, 0, 1024)
	_ = filepath.Walk("C:\\temp\\solution_directory", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, "java") {
			fileList = append(fileList, path)
		}
		return nil
	})
	c, err := NewMossSocketClient("java", "604014254")
	defer func(err error) {
		err = c.Close()
		if err != nil {
			println(err)
		}
	}(err)
	if err != nil {
		println(err)
		return
	}
	err = c.Run()
	for _, f := range fileList {
		err = c.UploadFile(f, false)
		if err != nil {
			println(err)
		}
	}
	err = c.SendQuery()
	if err != nil {
		println(err)
	}
	url := c.ResultURL
	fmt.Printf("%T, %v\n", url, url)
}
