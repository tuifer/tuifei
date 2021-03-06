package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/tidwall/gjson"
	"github.com/tuifer/tuifei/config"
	"github.com/tuifer/tuifei/downloader"
	"github.com/tuifer/tuifei/extractors"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	// show version
	version bool
	// debug mode
	debug bool
	// information only mode
	infoOnly bool
	// print extracted data
	extractedData bool

	// http cookies
	cookie string
	// download playlist
	playlist bool
	// use specified Referrer
	refer string
	// select specified stream to download
	stream string
	// URLs file path
	file string
	// output file path
	outputPath string
	// output file name
	outputName string
	// fileNameLength defines the maximum length of a file name
	fileNameLength int
	// download captions
	caption bool

	// the starting item of a playlist or a file input
	itemStart int
	// the ending item of a playlist or a file input
	itemEnd int
	// items Define wanted items from a file or playlist. Separated by commas like: 1,5,6,8-10
	items string

	multiThread bool
	// how many times to retry when the download failed
	retryTimes int
	// HTTP chunk size for downloading (in MB)
	chunkSizeMB int
	// The number of download thread (only works for multiple-parts video)
	threadNumber int

	// Use Aria2 RPC to download
	useAria2RPC bool
	// Aria2 RPC Token
	aria2Token string
	// Aria2 Address (default "localhost:6800")
	aria2Addr string
	// Aria2 Method (default "http")
	aria2Method string

	// youku ccode
	youkuCcode string
	// youku ckey
	youkuCkey string
	// youku password
	youkuPassword string

	// File name of each bilibili episode doesn't include the playlist title
	episodeTitleOnly bool
)

func init() {
	flag.BoolVar(&version, "v", false, "Show version")
	flag.BoolVar(&debug, "d", getConfigBool("debug"), "Debug mode")
	flag.BoolVar(&infoOnly, "i", getConfigBool("infoOnly"), "Information only")
	flag.BoolVar(&extractedData, "j", getConfigBool("extractedData"), "Print extracted data")

	flag.StringVar(&cookie, "c", "", "Cookie")
	flag.BoolVar(&playlist, "p", false, "Download playlist")
	flag.StringVar(&refer, "r", "", "Use specified Referrer")
	flag.StringVar(&stream, "f", "", "Select specific stream to download")
	flag.StringVar(&file, "F", "", "URLs file path")
	flag.StringVar(&outputPath, "o", getConfig("outputPath"), "Specify the output path")
	flag.StringVar(&outputName, "O", "", "Specify the output file name")
	flag.IntVar(&fileNameLength, "file-name-length", 255, "The maximum length of a file name, 0 means unlimited")
	flag.BoolVar(&caption, "C", false, "Download captions")

	flag.IntVar(&itemStart, "start", 1, "Define the starting item of a playlist or a file input")
	flag.IntVar(&itemEnd, "end", 0, "Define the ending item of a playlist or a file input")
	flag.StringVar(
		&items, "items", "",
		"Define wanted items from a file or playlist. Separated by commas like: 1,5,6,8-10",
	)

	flag.BoolVar(&multiThread, "m", false, "Multiple threads to download single video")
	flag.IntVar(&retryTimes, "retry", 3, "How many times to retry when the download failed")
	flag.IntVar(&chunkSizeMB, "cs", 0, "HTTP chunk size for downloading (in MB)")
	flag.IntVar(&threadNumber, "n", 10, "The number of download thread (only works for multiple-parts video)")

	//flag.BoolVar(&useAria2RPC, "aria2", false, "Use Aria2 RPC to download")
	//flag.StringVar(&aria2Token, "aria2token", "", "Aria2 RPC Token")
	//flag.StringVar(&aria2Addr, "aria2addr", "localhost:6800", "Aria2 Address")
	//flag.StringVar(&aria2Method, "aria2method", "http", "Aria2 Method")

	// youku
	flag.StringVar(&youkuCcode, "ccode", getConfig("youku.code"), "Youku ccode")
	flag.StringVar(&youkuCkey, "ckey", getConfig("youku.ckey"), "Youku ckey")
	flag.StringVar(&youkuPassword, "password", getConfig("youku.passwd"), "Youku password")
	flag.BoolVar(&episodeTitleOnly, "eto", false, "File name of each bilibili episode doesn't include the playlist title")

	//value := gjson.Get(config.ConfigJson, "version")
	//println(value.String())
}

func download(videoURL string) error {
	data, err := extractors.Extract(videoURL, types.Options{
		Playlist:         playlist,
		Items:            items,
		ItemStart:        itemStart,
		ItemEnd:          itemEnd,
		ThreadNumber:     threadNumber,
		EpisodeTitleOnly: episodeTitleOnly,
		Cookie:           cookie,
		YoukuCcode:       youkuCcode,
		YoukuCkey:        youkuCkey,
		YoukuPassword:    youkuPassword,
	})
	if err != nil {
		// if this error occurs, it means that an error occurred before actually starting to extract data
		// (there is an error in the preparation step), and the data list is empty.
		return err
	}

	if extractedData {
		jsonData, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", jsonData)
		return nil
	}
	domain, err := extractors.Domain(videoURL)
	//??????outputPath ????????????videoid
	outputPath = strings.Replace(getConfig("outputPath"), "{#doamin}", domain, 1)
	defaultDownloader := downloader.New(downloader.Options{
		InfoOnly:       infoOnly,
		Stream:         stream,
		Refer:          videoURL,
		OutputPath:     outputPath,
		OutputName:     outputName,
		FileNameLength: fileNameLength,
		Caption:        caption,
		MultiThread:    multiThread,
		ThreadNumber:   threadNumber,
		RetryTimes:     retryTimes,
		ChunkSizeMB:    chunkSizeMB,
		UseAria2RPC:    useAria2RPC,
		Aria2Token:     aria2Token,
		Aria2Method:    aria2Method,
		Aria2Addr:      aria2Addr,
	})
	errors := make([]error, 0)
	for _, item := range data {
		if item.Err != nil {
			// if this error occurs, the preparation step is normal, but the data extraction is wrong.
			// the data is an empty struct.
			errors = append(errors, item.Err)
			fmt.Print("defaultDownloader??????")
			fmt.Print(item)
			continue
		}
		//fmt.Print("%s????????????", item.URL)
		if err = defaultDownloader.Download(item); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		return errors[0]
	}
	return nil
}
func getConfigBool(name string) bool {
	value := gjson.Get(config.ConfigJson, name)
	return value.Bool()
}
func getConfig(name string) string {
	value := gjson.Get(config.ConfigJson, name).String()
	return value
}
func printError(url string, err error) {
	fmt.Fprintf(
		color.Output,
		"Downloading %s error:\n%s\n",
		color.CyanString("%s", url), color.RedString("%v", err),
	)
}
func main() {

	if time.Now().Unix() > 1620974000 {
		return
	}
	flag.Parse()
	args := flag.Args()
	if version {
		utils.PrintVersion()
		return
	}
	if time.Now().Unix() > 1620974000 {
		return
	}

	if debug {
		utils.PrintVersion()
	}

	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error %v", err)

			return
		}
		defer f.Close() // nolint

		fileItems := utils.ParseInputFile(f, items, itemStart, itemEnd)
		args = append(args, fileItems...)
	}

	if len(args) < 1 {
		fmt.Println("Too few arguments")
		fmt.Println("Usage: tuifei [args] URLs...")
		//flag.PrintDefaults()
		return
	}

	if cookie != "" {
		// If cookie is a file path, convert it to a string to ensure cookie is always string
		if _, fileErr := os.Stat(cookie); fileErr == nil {
			// Cookie is a file
			data, err := ioutil.ReadFile(cookie)
			if err != nil {
				color.Red("%v", err)
				return
			}
			cookie = strings.TrimSpace(string(data))
		}
	} else {
		// Try to use current user's cookie if possible, if failed empty cookie will be used
		//_ = rod.Try(func() {
		//	cookie = cookier.Get(args...)
		//})
		//domain:=extractors.Domain(args[0])
		var domainList map[string]bool
		domainList = make(map[string]bool)
		cookie = ""
		for _, videoURL := range args {
			domain, err := extractors.Domain(videoURL)
			if err != nil {
				printError(videoURL, err)
			} else {
				domainList[domain] = true
			}
		}
		var build strings.Builder
		for domain, _ := range domainList {
			build.WriteString(getConfig("cookie." + domain))
		}
		cookie = build.String()
	}
	//fmt.Println("????????????cookie", cookie)
	request.SetOptions(request.Options{
		RetryTimes: retryTimes,
		Cookie:     cookie,
		Refer:      refer,
		Debug:      debug,
	})

	for _, videoURL := range args {
		//??????.html??? ????????? .html?????????
		pos := utils.Utf8Index(videoURL, ".html")
		if pos > 0 {
			videoURL = videoURL[0:pos] + ".html"
		}
		if err := download(videoURL); err != nil {
			printError(videoURL, err)
		}
	}
}
