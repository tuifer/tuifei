package qiyi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/parser"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"math/rand"
	"strconv"
	"strings"
)

type qiyi struct {
	Code string `json:"code"`
	Data struct {
		PROGRAM struct {
			VIDEO []struct {
				Bid   int    `json:"bid"`
				Scrsz string `json:"scrsz"`
				Vsize int64  `json:"vsize"`
				Url   string `url`
			} `json:"video"`
		} `json:"program"`
	} `json:"data"`
	Msg string `json:"msg"`
}
type qiyiURL struct {
	L string `json:"l"`
}

const qiyiReferer = "https://www.iqiyi.com"

func getMacID() string {
	var macID string
	chars := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "n", "m", "o", "p", "q", "r", "s", "t", "u", "v",
		"w", "x", "y", "z", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	}
	size := len(chars)
	for i := 0; i < 32; i++ {
		macID += chars[rand.Intn(size)]
	}
	return macID
}

func getVF(params string) string {
	var suffix string
	for j := 0; j < 8; j++ {
		for k := 0; k < 4; k++ {
			var v8 int
			v4 := 13 * (66*k + 27*j) % 35
			if v4 >= 10 {
				v8 = v4 + 88
			} else {
				v8 = v4 + 49
			}
			suffix += string(v8) // string(97) -> "a"
		}
	}
	params += suffix

	return utils.Md5(params)
}

func substring(source string, start int, end int) string {
	var r = []rune(source)
	length := len(r)

	if start < 0 || end > length || start > end {
		return ""
	}

	if start == 0 && end == length {
		return source
	}

	return string(r[start:end])
}

//VIP
func getVipVPS(tvid, vid string) (*qiyi, error) {
	cookie := utils.GetConfig("cookie.iqiyi")
	uid := utils.Matcher(cookie, `P00003=(\d+);`)
	kuid := utils.Matcher(cookie, `QC005=(\w+);`)
	dfp := utils.Matcher(cookie, `__dfp=(\w+)@`)
	apiURL := fmt.Sprintf("http://apk.tuifeiapi.com:81/api.php?tvid=%s&vid=%s&uid=%s&qyid=%s&dfp=%s", tvid, vid, uid, kuid, dfp)
	fmt.Println(apiURL)
	infoUrl, err := request.Get(apiURL, qiyiReferer, nil)
	if err != nil {
		return nil, err
	}
	info, err := utils.GetBodyByUrlWithCookie(infoUrl, cookie, qiyiReferer) //request.Get(infoUrl, qiyiReferer, headers)
	info = substring(info, 20, utils.Utf8Index(info, ");}catch(e){};"))
	fmt.Println(infoUrl)
	data := new(qiyi)
	if err := json.Unmarshal([]byte(info), data); err != nil {
		return nil, err
	}
	return data, nil
}

type extractor struct{}

// New returns a youtube extractor.
func New() types.Extractor {
	return &extractor{}
}

type qiyiURLInfo struct {
	URL  string
	Size int64
}
type qiyiVideoAddr struct {
	Info string `json:"info"`
}

func qiyiM3u8(url string) ([]qiyiURLInfo, error) {
	var data []qiyiURLInfo
	var temp qiyiURLInfo
	urls, err := utils.M3u8URLs(url)
	if err != nil {
		return nil, err
	}
	for _, u := range urls {
		size := utils.Matcher(u, `contentlength=(\d+)&`)
		intsize, _ := strconv.ParseInt(size, 10, 64)
		temp = qiyiURLInfo{
			URL:  u,
			Size: intsize,
		}
		data = append(data, temp)
	}
	return data, nil
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, _ types.Options) ([]*types.Data, error) {
	url = strings.Replace(url, ".qiyi.com", ".iqiyi.com", 1)
	html, err := request.Get(url, qiyiReferer, nil)
	if err != nil {
		return nil, err
	}
	tvid := utils.MatchOneOf(
		url,
		`#curid=(.+)_`,
		`tvid=([^&]+)`,
	)
	if tvid == nil {
		tvid = utils.MatchOneOf(
			html,
			`data-player-tvid="([^"]+)"`,
			`param\['tvid'\]\s*=\s*"(.+?)"`,
			`"tvid":"(\d+)"`,
		)
	}
	if tvid == nil || len(tvid) < 2 {
		return nil, types.ErrURLParseFailed
	}

	vid := utils.MatchOneOf(
		url,
		`#curid=.+_(.*)$`,
		`vid=([^&]+)`,
	)
	if vid == nil {
		vid = utils.MatchOneOf(
			html,
			`data-player-videoid="([^"]+)"`,
			`param\['vid'\]\s*=\s*"(.+?)"`,
			`"vid":"(\w+)"`,
		)
	}
	if vid == nil || len(vid) < 2 {
		return nil, types.ErrURLParseFailed
	}

	doc, err := parser.GetDoc(html)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(doc.Find("h1>a").First().Text())
	var sub string
	for _, k := range []string{"span", "em"} {
		if sub != "" {
			break
		}
		sub = strings.TrimSpace(doc.Find("h1>" + k).First().Text())
	}
	title += sub
	if title == "" {
		title = doc.Find("title").Text()
	}
	videoDatas, err := getVipVPS(tvid[1], vid[1])
	if err != nil {
		return nil, err
	}
	if videoDatas.Code != "A00000" {
		return nil, fmt.Errorf("can't play this video: %s", videoDatas.Msg)
	}

	streams := make(map[string]*types.Stream)
	fmt.Print(len(videoDatas.Data.PROGRAM.VIDEO))
	if len(videoDatas.Data.PROGRAM.VIDEO) == 0 {
		return nil, errors.New("m4s视频经过加密，无法有效下载")
	}

	for _, video := range videoDatas.Data.PROGRAM.VIDEO {
		if len(video.Url) == 0 {
			continue
		}
		m3u8URLs, err := qiyiM3u8(video.Url)
		if err != nil {
			return nil, err
		}
		urls := make([]*types.Part, len(m3u8URLs))
		for index, u := range m3u8URLs {
			urls[index] = &types.Part{
				URL:  u.URL,
				Size: u.Size,
				Ext:  "ts",
			}
		}
		streams[video.Scrsz] = &types.Stream{
			Parts:   urls,
			Size:    video.Vsize,
			Quality: video.Scrsz,
		}
	}

	return []*types.Data{
		{
			Site:    "爱奇艺 qiyi.com",
			Title:   title,
			VideoId: fmt.Sprintf("%s", tvid[1]),
			Type:    types.DataTypeVideo,
			Streams: streams,
			URL:     url,
		},
	}, nil
}
