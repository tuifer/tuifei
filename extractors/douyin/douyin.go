package douyin

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"net/http"
	"strings"
	"time"
)

type data struct {
	ItemList []struct {
		Video struct {
			PlayAddr struct {
				Urllist []string `json:"url_list"`
			} `json:"play_addr"`
		} `json:"video"`
		Desc string `json:"desc"`
	} `json:"item_list"`
}

type extractor struct{}

// New returns a youtube extractor.
func New() types.Extractor {
	return &extractor{}
}
func GetRedirectUrl(sourceUrl string, headers map[string]string) (redirectUrl string, err error) {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //ssl证书报错问题
		DisableKeepAlives: false,                                 //关闭连接复用，因为后台连接过多最后会造成端口耗尽
		MaxIdleConns:      100,                                   //最大空闲连接数量
		IdleConnTimeout:   time.Duration(5 * time.Second),        //空闲连接超时时间
	}
	client := &http.Client{
		Timeout:   time.Duration(time.Second * 30),
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest("GET", sourceUrl, nil)
	if err != nil {
		return
	}
	request.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.70 Safari/537.36")

	if headers != nil {
		for key := range headers {
			request.Header.Set(key, headers[key])
		}
	}
	rsq, err := client.Do(request)
	if err != nil {
		return
	}
	statusCode := rsq.StatusCode
	if statusCode == 302 || statusCode == 301 {
		redirectUrl = rsq.Header.Get("Location")
		return
	} else {
		err = errors.New("request status not redirect")
		return
	}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option types.Options) ([]*types.Data, error) {
	videoIDs := utils.MatchOneOf(url, `/video/(\d+)/`)
	if len(videoIDs) == 0 {
		return nil, errors.New("unable to get video ID")
	}
	videoID := videoIDs[1]
	option.MyMain.LogAppend("抖音视频ID:" + videoID)
	apiDataString, err := request.Get(
		fmt.Sprintf("https://www.douyin.com/web/api/v2/aweme/iteminfo/?item_ids=%s", videoID),
		url, nil,
	)
	if err != nil {
		return nil, err
	}
	var apiData data
	if err = json.Unmarshal([]byte(apiDataString), &apiData); err != nil {
		return nil, err
	}
	fmt.Print(apiData.ItemList[0].Video.PlayAddr.Urllist)

	realURL := apiData.ItemList[0].Video.PlayAddr.Urllist[0]
	realURL = strings.Replace(realURL, "/playwm/", "/play/", 1)
	option.MyMain.LogAppend("开始下载无水印抖音视频")
	if realURL == "" || len(realURL) < 2 {
		return nil, types.ErrURLParseFailed
	}
	header := map[string]string{"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3 like Mac OS X) AppleWebKit/602.1.50 (KHTML, like Gecko) CriOS/56.0.2924.75 Mobile/14E5239e Safari/602.1"}

	loc, err := GetRedirectUrl(realURL, header)
	if err != nil {
		fmt.Println("获取转型地址失败")
	}
	size, err := request.Size(loc, url)
	if err != nil {
		return nil, err
	}
	urlData := &types.Part{
		URL:  loc,
		Size: size,
		Ext:  "mp4",
	}
	streams := map[string]*types.Stream{
		"default": {
			Parts: []*types.Part{urlData},
			Size:  size,
		},
	}

	return []*types.Data{
		{
			Site:    "抖音 douyin.com",
			Title:   apiData.ItemList[0].Desc,
			VideoId: videoID,
			Type:    types.DataTypeVideo,
			Streams: streams,
			URL:     url,
		},
	}, nil
}
