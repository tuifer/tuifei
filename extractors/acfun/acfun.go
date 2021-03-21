package acfun

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/parser"
	"github.com/tuifer/tuifei/utils"
	"net/url"
)

const (
	bangumiDataPattern = `window.pageInfo = window.bangumiData = (.*);
        window.qualityConfig =`
	qualityConfigPattern = "window.qualityConfig = (.*);"
	bangumiListPattern   = "window.bangumiList = (.*);"

	bangumiHTMLURL  = "https://www.acfun.cn/bangumi/aa%d_36188_%d"
	bangumiVideoId  = "%d_%d"
	bangumiVideoURL = "https://%s/mediacloud/acfun/acfun_video/hls/"

	referer = "https://www.acfun.cn"
)

var (
	videoType int
	videoId   string
)

type extractor struct{}

// New returns a new acfun bangumi extractor
func New() types.Extractor {
	return &extractor{}
}

func (e *extractor) Extract(URL string, option types.Options) ([]*types.Data, error) {
	html, err := utils.GetBodyByUrlWithCookie(URL, utils.GetConfig("cookie.acfun"), referer) //request.GetByte(URL, referer, nil)
	if err != nil {
		return nil, err
	}
	videoId = utils.Matcher(URL, `/v/ac(\d+)`) //普通视频
	if videoId != "" {
		videoType = 0
	} else {
		videoType = 1
	}
	if videoType == 1 { //https://www.acfun.cn/bangumi/aa6004596
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
				datas = append(datas, extractBangumi(concatURL(t), t, option))
			}()
		}
		wgp.Wait()
		return datas, nil
	} else { //https://www.acfun.cn/v/ac26823311
		if err != nil {
			return nil, err
		}
		datas := make([]*types.Data, 0)
		datas = append(datas, extractNomarl(html, videoId, URL))
		return datas, nil
	}

}

func concatURL(epData *episodeData) string {
	return fmt.Sprintf(bangumiHTMLURL, epData.BangumiID, epData.ItemID)
}
func extractNomarl(html string, videoId string, URL string) *types.Data {
	//resolvingNomarlData
	vInfo, err := resolvingNomarlData(html)
	if err != nil {
		return nil
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
		VideoId: videoId,
		Type:    types.DataTypeVideo,
		Streams: streams,
		URL:     URL,
	}
	return data
}
func resolvingNomarlData(html string) (*videoInfo, error) {
	bgData := &bangumiData{}
	vInfo := &videoInfo{}
	groups := utils.MatchOneOf(html, `window.pageInfo = window.videoInfo = (.+);`) //pattern.FindSubmatch(html)
	if groups == nil {
		return nil, types.ErrStringMatch
	}
	err := jsoniter.Unmarshal([]byte(groups[1]), bgData)
	if err != nil {
		return nil, err
	}
	err = jsoniter.UnmarshalFromString(bgData.CurrentVideoInfo.KsPlayJSON, vInfo)
	if err != nil {
		return nil, err
	}
	return vInfo, nil
}
func extractBangumi(URL string, epData *episodeData, option types.Options) *types.Data {
	var err error
	html, err := utils.GetBodyByUrlWithCookie(URL, utils.GetConfig("cookie.acfun"), referer) // request.GetByte(URL, referer, nil)
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
		VideoId: fmt.Sprintf(bangumiVideoId, epData.BangumiID, epData.ItemID),
		Type:    types.DataTypeVideo,
		Streams: streams,
		URL:     URL,
	}
	return data
}

func resolvingData(html string) (*bangumiData, *videoInfo, error) {
	bgData := &bangumiData{}
	vInfo := &videoInfo{}
	groups := utils.MatchOneOf(html, bangumiDataPattern) //pattern.FindSubmatch(html)
	if groups == nil {
		return nil, nil, types.ErrStringMatch
	}
	err := jsoniter.Unmarshal([]byte(groups[1]), bgData)
	if err != nil {
		return nil, nil, err
	}
	err = jsoniter.UnmarshalFromString(bgData.CurrentVideoInfo.KsPlayJSON, vInfo)
	if err != nil {
		return nil, nil, err
	}
	return bgData, vInfo, nil
}

func resolvingEpisodes(html string) (*episodeList, error) {
	list := &episodeList{}
	groups := utils.MatchOneOf(html, bangumiListPattern)
	err := jsoniter.Unmarshal([]byte(groups[1]), list)
	if err != nil {
		return nil, err
	}
	return list, nil
}
