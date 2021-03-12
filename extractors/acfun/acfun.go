package acfun

import (
	"fmt"
	"net/url"
	"regexp"

	jsoniter "github.com/json-iterator/go"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/parser"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
)

const (
	bangumiDataPattern   = "window.pageInfo = window.bangumiData = (.*);"
	qualityConfigPattern = "window.qualityConfig = (.*);"
	bangumiListPattern   = "window.bangumiList = (.*);"

	bangumiHTMLURL  = "https://www.acfun.cn/bangumi/aa%d_36188_%d"
	bangumiVideoId  = "acfun_%d_%d"
	bangumiVideoURL = "https://%s/mediacloud/acfun/acfun_video/hls/"

	referer = "https://www.acfun.cn"
	host    = "https://www.acfun.cn"
)

type extractor struct{}

// New returns a new acfun bangumi extractor
func New() types.Extractor {
	return &extractor{}
}

func (e *extractor) Extract(URL string, option types.Options) ([]*types.Data, error) {
	html, err := request.GetByte(URL, referer, nil)
	if err != nil {
		return nil, err
	}

	epDatas := make([]*episodeData, 0)

	if option.Playlist {
		list, err := resolvingEpisodes(html)
		if err != nil {
			return nil, err
		}
		items := utils.NeedDownloadList(option.Items, option.ItemStart, option.ItemEnd, len(list.Episodes))

		for _, item := range items {
			epDatas = append(epDatas, list.Episodes[item-1])
		}
	} else {
		bgData, _, err := resolvingData(html)
		if err != nil {
			return nil, err
		}
		epDatas = append(epDatas, &bgData.episodeData)
	}

	datas := make([]*types.Data, 0)

	wgp := utils.NewWaitGroupPool(option.ThreadNumber)
	for _, epData := range epDatas {
		t := epData
		wgp.Add()
		go func() {
			defer wgp.Done()
			option.VideoId = acfunVideoId(t)
			datas = append(datas, extractBangumi(concatURL(t), option))
		}()
	}
	wgp.Wait()
	return datas, nil
}
func acfunVideoId(epData *episodeData) string {
	return fmt.Sprintf(bangumiVideoId, epData.BangumiID, epData.ItemID)
}
func concatURL(epData *episodeData) string {
	return fmt.Sprintf(bangumiHTMLURL, epData.BangumiID, epData.ItemID)
}

func extractBangumi(URL string, option types.Options) *types.Data {
	var err error
	html, err := request.GetByte(URL, referer, nil)
	if err != nil {
		return types.EmptyData(URL, err)
	}

	_, vInfo, err := resolvingData(html)
	if err != nil {
		return types.EmptyData(URL, err)
	}

	streams := make(map[string]*types.Stream)

	for _, stm := range vInfo.AdaptationSet[0].Streams {
		m3u8URL, err := url.Parse(stm.URL)
		if err != nil {
			return types.EmptyData(URL, err)
		}

		urls, err := utils.M3u8URLs(m3u8URL.String())
		if err != nil {

			m3u8URL, err = url.Parse(stm.URL)
			if err != nil {
				return types.EmptyData(URL, err)
			}

			urls, err = utils.M3u8URLs(stm.BackURL)
			if err != nil {
				return types.EmptyData(URL, err)
			}
		}

		// There is no size information in the m3u8 file and the calculation will take too much time, just ignore it.
		parts := make([]*types.Part, 0)
		for _, u := range urls {
			parts = append(parts, &types.Part{
				URL: u,
				Ext: "ts",
			})
		}
		streams[stm.QualityLabel] = &types.Stream{
			ID:      stm.QualityType,
			Parts:   parts,
			Quality: stm.QualityType,
			NeedMux: false,
		}
	}

	doc, err := parser.GetDoc(string(html))
	if err != nil {
		return types.EmptyData(URL, err)
	}
	data := &types.Data{
		Site:    "AcFun acfun.cn",
		Title:   parser.Title(doc),
		VideoId: option.VideoId,
		Type:    types.DataTypeVideo,
		Streams: streams,
		URL:     URL,
	}
	return data
}

func resolvingData(html []byte) (*bangumiData, *videoInfo, error) {
	bgData := &bangumiData{}
	vInfo := &videoInfo{}

	pattern, _ := regexp.Compile(bangumiDataPattern)

	groups := pattern.FindSubmatch(html)
	err := jsoniter.Unmarshal(groups[1], bgData)
	if err != nil {
		return nil, nil, err
	}

	err = jsoniter.UnmarshalFromString(bgData.CurrentVideoInfo.KsPlayJSON, vInfo)
	if err != nil {
		return nil, nil, err
	}
	return bgData, vInfo, nil
}

func resolvingEpisodes(html []byte) (*episodeList, error) {
	list := &episodeList{}
	pattern, _ := regexp.Compile(bangumiListPattern)

	groups := pattern.FindSubmatch(html)
	err := jsoniter.Unmarshal(groups[1], list)
	if err != nil {
		return nil, err
	}
	return list, nil
}
