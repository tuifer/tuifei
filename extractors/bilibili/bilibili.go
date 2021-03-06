package bilibili

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/parser"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
)

const (
	bilibiliAPI        = "https://api.bilibili.com/x/player/playurl?"
	bilibiliBangumiAPI = "https://api.bilibili.com/pgc/player/web/playurl?"
	bilibiliTokenAPI   = "https://api.bilibili.com/x/player/playurl/token?"
)

const referer = "https://www.bilibili.com"

var utoken string

func genAPI(aid, cid, quality int, bvid string, bangumi bool, cookie string) (string, error) {
	var (
		err        error
		baseAPIURL string
		params     string
	)
	if cookie != "" && utoken == "" {
		utoken, err = request.Get(
			fmt.Sprintf("%said=%d&cid=%d", bilibiliTokenAPI, aid, cid),
			referer,
			nil,
		)
		if err != nil {
			return "", err
		}
		var t token
		err = json.Unmarshal([]byte(utoken), &t)
		if err != nil {
			return "", err
		}
		if t.Code != 0 {
			return "", fmt.Errorf("cookie error: %s", t.Message)
		}
		utoken = t.Data.Token
	}
	var api string
	if bangumi {
		// The parameters need to be sorted by name
		// qn=0 flag makes the CDN address different every time
		// quality=120(4k) is the highest quality so far
		params = fmt.Sprintf(
			"cid=%d&bvid=%s&qn=%d&type=&otype=json&fourk=1&fnver=0&fnval=16",
			cid, bvid, quality,
		)
		baseAPIURL = bilibiliBangumiAPI
	} else {
		params = fmt.Sprintf(
			"avid=%d&cid=%d&bvid=%s&qn=%d&type=&otype=json&fourk=1&fnver=0&fnval=16",
			aid, cid, bvid, quality,
		)
		baseAPIURL = bilibiliAPI
	}
	api = baseAPIURL + params
	// bangumi utoken also need to put in params to sign, but the ordinary video doesn't need
	if !bangumi && utoken != "" {
		api = fmt.Sprintf("%s&utoken=%s", api, utoken)
	}
	return api, nil
}

func genParts(dashData *dashInfo, quality int, referer string) ([]*types.Part, error) {
	parts := make([]*types.Part, 1)
	if dashData.Streams.Audio == nil {
		url := dashData.DURL[0].URL
		_, ext, err := utils.GetNameAndExt(url)
		if err != nil {
			return nil, err
		}
		parts[0] = &types.Part{
			URL:  url,
			Size: 0,
			Ext:  ext,
		}

	} else {

		checked := false
		for _, stream := range dashData.Streams.Video {
			if stream.ID == quality {
				//s, err := request.Size(stream.BaseURL, referer)
				//if err != nil {
				//	return nil, err
				//}
				//fmt.Println("%s ?????????%s",stream.BaseURL,strconv.Itoa(int(s)))
				parts[0] = &types.Part{
					URL:  stream.BaseURL,
					Size: 0,
					Ext:  "mp4",
				}
				checked = true
				break
			}
		}
		if !checked {
			return nil, nil
		}
	}
	return parts, nil
}

type bilibiliOptions struct {
	url      string
	html     string
	bangumi  bool
	aid      int
	cid      int
	bvid     string
	page     int
	subtitle string
}

func extractBangumi(url, html string, extractOption types.Options) ([]*types.Data, error) {
	dataString := utils.MatchOneOf(html, `window.__INITIAL_STATE__=(.+?);\(function`)[1]
	var data bangumiData
	err := json.Unmarshal([]byte(dataString), &data)
	if err != nil {
		return nil, err
	}
	if !extractOption.Playlist {
		aid := data.EpInfo.Aid
		cid := data.EpInfo.Cid
		bvid := data.EpInfo.BVid
		if aid <= 0 || cid <= 0 || bvid == "" {
			aid = data.EpList[0].Aid
			cid = data.EpList[0].Cid
			bvid = data.EpList[0].BVid
		}
		options := bilibiliOptions{
			url:     url,
			html:    html,
			bangumi: true,
			aid:     aid,
			cid:     cid,
			bvid:    bvid,
		}
		return []*types.Data{bilibiliDownload(options, extractOption)}, nil
	}

	// handle bangumi playlist
	needDownloadItems := utils.NeedDownloadList(extractOption.Items, extractOption.ItemStart, extractOption.ItemEnd, len(data.EpList))
	extractedData := make([]*types.Data, len(needDownloadItems))
	wgp := utils.NewWaitGroupPool(extractOption.ThreadNumber)
	dataIndex := 0
	for index, u := range data.EpList {
		if !utils.ItemInSlice(index+1, needDownloadItems) {
			continue
		}
		wgp.Add()
		id := u.EpID
		if id == 0 {
			id = u.ID
		}
		// html content can't be reused here
		options := bilibiliOptions{
			url:     fmt.Sprintf("https://www.bilibili.com/bangumi/play/ep%d", id),
			bangumi: true,
			aid:     u.Aid,
			cid:     u.Cid,
			bvid:    u.BVid,
		}
		go func(index int, options bilibiliOptions, extractedData []*types.Data) {
			defer wgp.Done()
			extractedData[index] = bilibiliDownload(options, extractOption)
		}(dataIndex, options, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

func getMultiPageData(html string) (*multiPage, error) {
	var data multiPage
	multiPageDataString := utils.MatchOneOf(
		html, `window.__INITIAL_STATE__=(.+?);\(function`,
	)
	if multiPageDataString == nil {
		return &data, errors.New("this page has no playlist")
	}
	err := json.Unmarshal([]byte(multiPageDataString[1]), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
func getPageData(html string) (*BiliData, error) {
	var data BiliData
	multiPageDataString := utils.MatchOneOf(
		html, `window.__playinfo__=(.+?)</script><script>window.__INITIAL_STATE`,
	)
	if multiPageDataString == nil {
		return &data, errors.New("this page has no playlist")
	}
	err := json.Unmarshal([]byte(multiPageDataString[1]), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
func getPart(dash *Dash, quality int) ([]*types.Part, error) {
	parts := make([]*types.Part, 1)
	if dash.Audio == nil {
		url := dash.Video[0].Url
		_, ext, err := utils.GetNameAndExt(url)
		if err != nil {
			return nil, err
		}
		parts[0] = &types.Part{
			URL:  url,
			Size: 0,
			Ext:  ext,
		}
	} else {

		checked := false
		for _, stream := range dash.Video {
			if stream.Id == quality {
				parts[0] = &types.Part{
					URL:  stream.Url,
					Size: 0,
					Ext:  "mp4",
				}
				checked = true
				break
			}
		}
		if !checked {
			return nil, nil
		}
	}
	return parts, nil
}
func extractNormalVideo(url string, html string, options types.Options) ([]*types.Data, error) {
	pageDatas, err := getMultiPageData(html)
	pageData, err := getPageData(html)
	if err != nil {
		return nil, err
	}
	dashData := pageData.Data.Dash
	var audioPart *types.Part
	if dashData.Audio != nil {
		// Get audio part
		var audioID int
		audios := map[int]string{}
		bandwidth := 0
		for _, stream := range dashData.Audio {
			if stream.Bandwidth > bandwidth {
				audioID = stream.Id
				bandwidth = stream.Bandwidth
			}
			audios[stream.Id] = stream.Url
		}
		s, err := request.Size(audios[audioID], referer)
		if err != nil {
			return nil, err
		}
		audioPart = &types.Part{
			URL:  audios[audioID],
			Size: s,
			Ext:  "m4a",
		}
	}
	streams := make(map[string]*types.Stream, len(pageData.Data.Formats))
	for _, q := range pageData.Data.Formats {
		if _, ok := streams[strconv.Itoa(q.Quality)]; ok {
			continue
		}
		parts, err := getPart(&dashData, q.Quality)
		if parts == nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		if audioPart != nil {
			parts = append(parts, audioPart)
		}
		var size int64
		for _, part := range parts {
			size += part.Size
		}
		streams[strconv.Itoa(q.Quality)] = &types.Stream{
			Parts:   parts,
			Size:    size,
			Quality: qualityString[q.Quality],
		}
		if audioPart != nil {
			streams[strconv.Itoa(q.Quality)].NeedMux = true
		}
	}
	page := pageDatas.VideoData.Pages[0]
	extractedData := make([]*types.Data, 1)
	extractedData[0] = &types.Data{
		Site:    "???????????? bilibili.com",
		Title:   "title",
		VideoId: strconv.Itoa(page.Cid),
		Type:    types.DataTypeVideo,
		Streams: streams,
		Caption: &types.Part{
			URL: fmt.Sprintf("https://comment.bilibili.com/%d.xml", page.Cid),
			Ext: "xml",
		},
		URL: url,
	}
	return extractedData, nil
}

type extractor struct{}

// New returns a youtube extractor.
func New() types.Extractor {
	return &extractor{}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option types.Options) ([]*types.Data, error) {
	var err error
	html, err := request.Get(url, referer, nil)
	if err != nil {
		return nil, err
	}

	// set thread number to 1 manually to avoid http 412 error
	option.ThreadNumber = 1
	//fmt.Printf("Warning: Multi thread download is no longer supported by BiliBili, use single thread instead.\n")

	if strings.Contains(url, "bangumi") {
		// handle bangumi
		return extractBangumi(url, html, option)
	}
	// handle normal video
	return extractNormalVideo(url, html, option)
}

// bilibiliDownload is the download function for a single URL
func bilibiliDownload(options bilibiliOptions, extractOption types.Options) *types.Data {
	var (
		err  error
		html string
	)
	if options.html != "" {
		// reuse html string, but this can't be reused in case of playlist
		html = options.html
	} else {
		html, err = request.Get(options.url, referer, nil)
		if err != nil {
			return types.EmptyData(options.url, err)
		}
	}

	// Get "accept_quality" and "accept_description"
	// "accept_description":["?????? 1080P","?????? 720P","?????? 480P","?????? 360P"],
	// "accept_quality":[120,112,80,48,32,16],
	api, err := genAPI(options.aid, options.cid, 120, options.bvid, options.bangumi, extractOption.Cookie)
	if err != nil {
		return types.EmptyData(options.url, err)
	}
	jsonString, err := request.Get(api, referer, nil)
	if err != nil {
		return types.EmptyData(options.url, err)
	}
	//extractOption.MyMain.LogAppend("jsonString :"+jsonString)
	var data dash
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return types.EmptyData(options.url, err)
	}
	var dashData dashInfo
	if data.Data.Description == nil {
		dashData = data.Result
	} else {
		dashData = data.Data
	}

	var audioPart *types.Part
	if dashData.Streams.Audio != nil {
		// Get audio part
		var audioID int
		audios := map[int]string{}
		bandwidth := 0
		for _, stream := range dashData.Streams.Audio {
			if stream.Bandwidth > bandwidth {
				audioID = stream.ID
				bandwidth = stream.Bandwidth
			}
			audios[stream.ID] = stream.BaseURL
		}
		//s, err := request.Size(audios[audioID], referer)
		//if err != nil {
		//	return types.EmptyData(options.url, err)
		//}
		audioPart = &types.Part{
			URL:  audios[audioID],
			Size: 0,
			Ext:  "m4a",
		}
	}

	streams := make(map[string]*types.Stream, len(dashData.Quality))

	for _, q := range dashData.Quality {
		// Avoid duplicate streams
		if _, ok := streams[strconv.Itoa(q)]; ok {
			continue
		}

		parts, err := genParts(&dashData, q, options.url)
		if parts == nil {
			continue
		}
		if err != nil {
			return types.EmptyData(options.url, err)
		}
		if audioPart != nil {
			parts = append(parts, audioPart)
		}
		var size int64
		for _, part := range parts {
			size += part.Size
		}
		streams[strconv.Itoa(q)] = &types.Stream{
			Parts:   parts,
			Size:    size,
			Quality: qualityString[q],
		}
		if audioPart != nil {
			streams[strconv.Itoa(q)].NeedMux = true
		}
	}

	// get the title
	doc, err := parser.GetDoc(html)
	if err != nil {
		return types.EmptyData(options.url, err)
	}
	title := parser.Title(doc)
	if options.subtitle != "" {
		if extractOption.EpisodeTitleOnly {
			title = fmt.Sprintf("P%d %s", options.page, options.subtitle)
		} else {
			title = fmt.Sprintf("%s P%d %s", title, options.page, options.subtitle)
		}
	}

	return &types.Data{
		Site:    "???????????? bilibili.com",
		Title:   title,
		VideoId: strconv.Itoa(options.cid),
		Type:    types.DataTypeVideo,
		Streams: streams,
		Caption: &types.Part{
			URL: fmt.Sprintf("https://comment.bilibili.com/%d.xml", options.cid),
			Ext: "xml",
		},
		URL: options.url,
	}
}
