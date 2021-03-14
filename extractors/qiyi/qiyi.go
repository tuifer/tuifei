package qiyi

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tuifer/tuifei/config"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/parser"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type qiyi struct {
	Code string `json:"code"`
	Data struct {
		VP struct {
			Du  string `json:"du"`
			Tkl []struct {
				Vs []struct {
					Bid   int    `json:"bid"`
					Scrsz string `json:"scrsz"`
					Vsize int64  `json:"vsize"`
					Fs    []struct {
						L string `json:"l"`
						B int64  `json:"b"`
					} `json:"fs"`
				} `json:"vs"`
			} `json:"tkl"`
		} `json:"vp"`
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
func getConfig(name string) string {
	value := gjson.Get(config.ConfigJson, name).String()
	return value
}
func matcher(str string, reg string) string {
	rePart := utils.MatchOneOf(str, reg)
	if len(rePart) > 1 {
		return rePart[1]
	} else {
		return ""
	}
}

//VIP
func getVipVPS(tvid, vid string) (*qiyi, error) {
	//根据cookie 获取qiyi的dfp 和uid 还有kuid
	cookie := getConfig("cookie.iqiyi")
	uid := matcher(cookie, `P00003=(\d+);`)
	kuid := matcher(cookie, `QC005=(\w+);`)
	dfp := matcher(cookie, `__dfp=(\w+)@`)
	apiURL := fmt.Sprintf("http://apk.tuifeiapi.com:81/api.php?tvid=%s&vid=%s&uid=%s&qyid=%s&dfp=%s", tvid, vid, uid, kuid, dfp)
	fmt.Println(apiURL)
	headers := map[string]string{
		"Cookie":     cookie,
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.104 Safari/537.36",
	}
	infoUrl, err := request.Get(apiURL, qiyiReferer, nil)
	if err != nil {
		return nil, err
	}
	info, err := request.Get(infoUrl, qiyiReferer, headers)
	fmt.Println(info)
	data := new(qiyi)
	if err := json.Unmarshal([]byte(info), data); err != nil {
		return nil, err
	}
	return data, nil
}
func getVPS(tvid, vid string) (*qiyi, error) {
	t := time.Now().Unix() * 1000
	host := "http://cache.video.iqiyi.com"
	params := fmt.Sprintf(
		"/vps?tvid=%s&vid=%s&v=0&qypid=%s_12&src=01012001010000000000&t=%d&k_tag=1&k_uid=%s&rs=1",
		tvid, vid, tvid, t, getMacID(),
	)
	vf := getVF(params)
	apiURL := fmt.Sprintf("%s%s&vf=%s", host, params, vf)
	fmt.Println(apiURL)
	info, err := request.Get(apiURL, qiyiReferer, nil)
	if err != nil {
		return nil, err
	}
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
	urlPrefix := videoDatas.Data.VP.Du
	for _, video := range videoDatas.Data.VP.Tkl[0].Vs {
		urls := make([]*types.Part, len(video.Fs))
		for index, v := range video.Fs {
			realURLData, err := request.Get(urlPrefix+v.L, qiyiReferer, nil)
			if err != nil {
				return nil, err
			}
			var realURL qiyiURL
			if err = json.Unmarshal([]byte(realURLData), &realURL); err != nil {
				return nil, err
			}
			_, ext, err := utils.GetNameAndExt(realURL.L)
			if err != nil {
				return nil, err
			}
			urls[index] = &types.Part{
				URL:  realURL.L,
				Size: v.B,
				Ext:  ext,
			}
		}
		streams[strconv.Itoa(video.Bid)] = &types.Stream{
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
