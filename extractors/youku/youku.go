package youku

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idoubi/goz"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"math/rand"
	netURL "net/url"
	"strconv"
	"strings"
	"time"
)

type errorData struct {
	Note string `json:"note"`
	Code int    `json:"code"`
}

type segs struct {
	Size int64  `json:"size"`
	URL  string `json:"cdn_url"`
}

type stream struct {
	Size      int64  `json:"size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Segs      []segs `json:"segs"`
	Type      string `json:"stream_type"`
	AudioLang string `json:"audio_lang"`
	M3u8Url   string `json:"m3u8_url"`
}

type youkuVideo struct {
	Title string `json:"title"`
}

type youkuShow struct {
	Title string `json:"title"`
}
type data struct {
	Video  youkuVideo `json:"video"`
	Stream []stream   `json:"stream"`
	Show   youkuShow  `json:"show"`
	Error  errorData  `json:"error"`
}

type youkuData struct {
	Data struct {
		Data data `json:"data"`
	} `json:"data"`
}

const youkuReferer = "https://v.youku.com"
const youkuTokenApi = "http://acs.youku.com/h5/mtop.youku.play.ups.appinfo.get/1.1/?appKey=24679788&api=mtop.youku.play.ups.appinfo.get"
const youkuUrl = "http://acs.youku.com/h5/mtop.youku.play.ups.appinfo.get/1.1/?"
const youkuApi = "mtop.youku.play.ups.appinfo.get"

func getAudioLang(lang string) string {
	var youkuAudioLang = map[string]string{
		"guoyu": "国语",
		"ja":    "日语",
		"yue":   "粤语",
	}
	translate, ok := youkuAudioLang[lang]
	if !ok {
		return lang
	}
	return translate
}

// https://g.alicdn.com/player/ykplayer/0.5.61/youku-player.min.js
// {"0505":"interior","050F":"interior","0501":"interior","0502":"interior","0503":"interior","0510":"adshow","0512":"BDskin","0590":"BDskin"}

// var ccodes = []string{"0510", "0502", "0507", "0508", "0512", "0513", "0514", "0503", "0590"}

func genCookie(cookie string) (string, error) {
	cli := goz.NewClient()
	resp, err := cli.Get(youkuTokenApi, goz.Options{
		Headers: map[string]interface{}{
			"Cookie":     cookie,
			"User-Agent": utils.AGENT,
		},
	})
	if err != nil {
		return "", err
	}
	args := resp.GetHeader("Set-Cookie")
	var str strings.Builder
	for _, v := range args {
		pos := utils.Utf8Index(v, ";")
		str.WriteString(v[0:pos] + ";")
	}
	return str.String(), nil
}

func youkuUps(vid string, option types.Options) (*youkuData, error) {
	var (
		url   string
		utid  string
		utids []string
		data  youkuData
	)
	cookie := utils.GetConfig("cookie.youku")
	newCookie, _ := genCookie(cookie)
	newToken := utils.Matcher(newCookie, `_m_h5_tk=(\w+)_`)
	if strings.Contains(cookie, "cna") {
		utids = utils.MatchOneOf(cookie, `cna=(.+?);`, `cna\s+(.+?)\s`, `cna\s+(.+?)$`)
	} else {
		headers, err := request.Headers("http://log.mmstat.com/eg.js", youkuReferer)
		if err != nil {
			return nil, err
		}
		setCookie := headers.Get("Set-Cookie")
		utids = utils.MatchOneOf(setCookie, `cna=(.+?);`)
	}
	if utids == nil || len(utids) < 2 {
		return nil, types.ErrURLParseFailed
	}
	utid = utids[1]

	appKey := utils.GetConfig("youku.appkey")
	client_ts := fmt.Sprintf("%d", time.Now().Unix())
	version := utils.GetConfig("youku.version")
	postStr := fmt.Sprintf(`{"steal_params":"{\"ccode\":\"%s\",\"client_ip\":\"%s\",\"utid\":\"%s\",\"client_ts\":%s,\"version\":\"%s\",\"ckey\":\"%s\"}","biz_params":"{\"vid\":\"%s\",\"play_ability\":%s,\"drm_type\":1,\"key_index\":\"web01\",\"encryptR_client\":\"%s\",\"preferClarity\":5,\"extag\":\"EXT-X-PRIVINF\",\"master_m3u8\":1,\"media_type\":\"standard,subtitle\",\"app_ver\":\"2.1.63\"}","ad_params":"{\"vs\":\"1.0\",\"pver\":\"2.1.63\",\"sver\":\"1.0\",\"site\":1,\"aw\":\"w\",\"fu\":0,\"d\":\"0\",\"bt\":\"pc\",\"os\":\"win\",\"osv\":\"10\",\"dq\":\"auto\",\"atm\":\"\",\"partnerid\":\"null\",\"wintype\":\"interior\",\"isvert\":0,\"vip\":0,\"emb\":\"%s\",\"p\":1,\"rst\":\"mp4\",\"needbf\":2,\"avs\":\"1.0\"}"}`,
		utils.GetConfig("youku.code"), utils.GetConfig("youku.client_ip"), utid, client_ts, version, utils.GetConfig("youku.ckey"), vid, utils.GetConfig("youku.play_ability"), utils.GetConfig("youku.encryptR_client"), utils.GetConfig("youku.emb"),
	)
	signStr := fmt.Sprintf("%s&%s&%s&%s", newToken, client_ts, appKey, postStr)

	sign := utils.Md5(signStr)
	referUrl := fmt.Sprintf("https://v.youku.com/v_show/id_%s.html", vid)

	for _, ccode := range []string{option.YoukuCcode} {
		if ccode == "0103010102" {
			utid = generateUtdid()
		}
		url = fmt.Sprintf("%sjsv=2.5.8&appKey=%s&t=%s&sign=%s&api=%s&data=%s", youkuUrl, appKey, client_ts, sign, youkuApi, netURL.QueryEscape(postStr))
		if option.YoukuPassword != "" {
			url = fmt.Sprintf("%s&password=%s", url, option.YoukuPassword)
		}
		fmt.Println(url)
		//fmt.Println(url)
		html, err := utils.GetBodyByUrlWithCookie(url, newCookie+cookie, referUrl)
		fmt.Println(html)
		if err != nil {
			return nil, err
		}
		// data must be emptied before reassignment, otherwise it will contain the previous value(the 'error' data)
		data = youkuData{}
		if err = json.Unmarshal([]byte(html), &data); err != nil {
			return nil, err
		}
		if data.Data.Data.Error == (errorData{}) {
			return &data, nil
		}
	}
	return &data, nil
}

func getBytes(val int32) []byte {
	var buff bytes.Buffer
	binary.Write(&buff, binary.BigEndian, val) // nolint
	return buff.Bytes()
}

func hashCode(s string) int32 {
	var result int32
	for _, c := range s {
		result = result*0x1f + c
	}
	return result
}

func hmacSha1(key []byte, msg []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(msg) // nolint
	return mac.Sum(nil)
}

func generateUtdid() string {
	timestamp := int32(time.Now().Unix())
	var buffer bytes.Buffer
	buffer.Write(getBytes(timestamp - 60*60*8))
	buffer.Write(getBytes(rand.Int31()))
	buffer.WriteByte(0x03)
	buffer.WriteByte(0x00)
	imei := fmt.Sprintf("%d", rand.Int31())
	buffer.Write(getBytes(hashCode(imei)))
	data := hmacSha1([]byte("d6fc3a4a06adbde89223bvefedc24fecde188aaa9161"), buffer.Bytes())
	buffer.Write(getBytes(hashCode(base64.StdEncoding.EncodeToString(data))))
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

type youkuURLInfo struct {
	URL  string
	Size int64
}

func youkuM3u8(url string) ([]youkuURLInfo, error) {
	var data []youkuURLInfo
	var temp youkuURLInfo
	m3u8String, err := request.Get(url, url, nil)
	if err != nil {
		return nil, err
	}
	urls, err := utils.M3u8UrlByStr(url, m3u8String)
	if err != nil {
		return nil, err
	}
	sizes := utils.MatchAll(m3u8String, `#EXT-X-PRIVINF:FILESIZE=(\d+)`)
	for index, u := range urls {
		size, err := strconv.ParseInt(sizes[index][1], 10, 64)
		if err != nil {
			return nil, err
		}
		temp = youkuURLInfo{
			URL:  u,
			Size: size,
		}
		data = append(data, temp)
	}
	return data, nil
}
func genData(youkuData data) map[string]*types.Stream {
	var (
		streamString string
		quality      string
	)
	streams := make(map[string]*types.Stream, len(youkuData.Stream))
	for _, stream := range youkuData.Stream {
		if stream.AudioLang == "default" {
			streamString = stream.Type
			quality = fmt.Sprintf(
				"%s %dx%d", stream.Type, stream.Width, stream.Height,
			)
		} else {
			streamString = fmt.Sprintf("%s-%s", stream.Type, stream.AudioLang)
			quality = fmt.Sprintf(
				"%s %dx%d %s", stream.Type, stream.Width, stream.Height,
				getAudioLang(stream.AudioLang),
			)
		}

		ext := strings.Split(
			strings.Split(stream.Segs[0].URL, "?")[0],
			".",
		)
		urls := make([]*types.Part, len(stream.Segs))
		for index, data := range stream.Segs {
			urls[index] = &types.Part{
				URL:  data.URL,
				Size: data.Size,
				Ext:  ext[len(ext)-1],
			}
		}
		streams[streamString] = &types.Stream{
			Parts:   urls,
			Size:    stream.Size,
			Quality: quality,
		}
	}
	return streams
}

type extractor struct{}

// New returns a youtube extractor.
func New() types.Extractor {
	return &extractor{}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option types.Options) ([]*types.Data, error) {
	vids := utils.MatchOneOf(
		url, `id_(.+?)\.html`, `id_(.+)`,
	)
	if vids == nil || len(vids) < 2 {
		return nil, types.ErrURLParseFailed
	}
	vid := vids[1]

	youkuData, err := youkuUps(vid, option)
	if err != nil {
		return nil, err
	}
	if youkuData.Data.Data.Error.Code != 0 {
		return nil, errors.New(youkuData.Data.Data.Error.Note)
	}
	streams := genData(youkuData.Data.Data)
	var title string
	if youkuData.Data.Data.Show.Title == "" || strings.Contains(
		youkuData.Data.Data.Video.Title, youkuData.Data.Data.Show.Title,
	) {
		title = youkuData.Data.Data.Video.Title
	} else {
		title = fmt.Sprintf("%s %s", youkuData.Data.Data.Show.Title, youkuData.Data.Data.Video.Title)
	}
	return []*types.Data{
		{
			Site:    "优酷 youku.com",
			Title:   title,
			VideoId: vid,
			Type:    types.DataTypeVideo,
			Streams: streams,
			URL:     url,
		},
	}, nil
}
