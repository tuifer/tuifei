package config

import (
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"os"
	"strings"
)

// FakeHeaders fake http headers
var FakeHeaders = map[string]string{
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Charset":  "UTF-8,*;q=0.5",
	"Accept-Encoding": "gzip,deflate,sdch",
	"Accept-Language": "en-US,en;q=0.8",
	"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.81 Safari/537.36",
}
var ConfigJson string

//从本地读取Json 配置
func init() {
	tuifer := "./tuifer.json"
	if _, fileErr := os.Stat(tuifer); fileErr == nil {
		// Cookie is a file
		data, err := ioutil.ReadFile(tuifer)
		if err != nil {
			color.Red("%v", err)
			return
		}
		ConfigJson = strings.TrimSpace(string(data))
		fmt.Println("cookie 配置内容", ConfigJson)
	}
}
