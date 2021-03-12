package extractors

import (
	"net/url"
	"strings"

	"github.com/tuifer/tuifei/extractors/acfun"
	"github.com/tuifer/tuifei/extractors/bilibili"
	"github.com/tuifer/tuifei/extractors/douyin"
	"github.com/tuifer/tuifei/extractors/douyu"
	"github.com/tuifer/tuifei/extractors/facebook"
	"github.com/tuifer/tuifei/extractors/geekbang"
	"github.com/tuifer/tuifei/extractors/haokan"
	"github.com/tuifer/tuifei/extractors/instagram"
	"github.com/tuifer/tuifei/extractors/iqiyi"
	"github.com/tuifer/tuifei/extractors/mgtv"
	"github.com/tuifer/tuifei/extractors/miaopai"
	"github.com/tuifer/tuifei/extractors/netease"
	"github.com/tuifer/tuifei/extractors/pixivision"
	"github.com/tuifer/tuifei/extractors/pornhub"
	"github.com/tuifer/tuifei/extractors/qq"
	"github.com/tuifer/tuifei/extractors/tangdou"
	"github.com/tuifer/tuifei/extractors/tiktok"
	"github.com/tuifer/tuifei/extractors/tumblr"
	"github.com/tuifer/tuifei/extractors/twitter"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/extractors/udn"
	"github.com/tuifer/tuifei/extractors/universal"
	"github.com/tuifer/tuifei/extractors/vimeo"
	"github.com/tuifer/tuifei/extractors/weibo"
	"github.com/tuifer/tuifei/extractors/xvideos"
	"github.com/tuifer/tuifei/extractors/yinyuetai"
	"github.com/tuifer/tuifei/extractors/youku"
	"github.com/tuifer/tuifei/extractors/youtube"
	"github.com/tuifer/tuifei/utils"
)

var extractorMap map[string]types.Extractor

func init() {
	douyinExtractor := douyin.New()
	youtubeExtractor := youtube.New()

	extractorMap = map[string]types.Extractor{
		"": universal.New(), // universal extractor

		"douyin":     douyinExtractor,
		"iesdouyin":  douyinExtractor,
		"bilibili":   bilibili.New(),
		"pixivision": pixivision.New(),
		"youku":      youku.New(),
		"youtube":    youtubeExtractor,
		"youtu":      youtubeExtractor, // youtu.be
		"iqiyi":      iqiyi.New(),
		"mgtv":       mgtv.New(),
		"tangdou":    tangdou.New(),
		"tumblr":     tumblr.New(),
		"vimeo":      vimeo.New(),
		"facebook":   facebook.New(),
		"douyu":      douyu.New(),
		"miaopai":    miaopai.New(),
		"163":        netease.New(),
		"weibo":      weibo.New(),
		"instagram":  instagram.New(),
		"twitter":    twitter.New(),
		"qq":         qq.New(),
		"yinyuetai":  yinyuetai.New(),
		"geekbang":   geekbang.New(),
		"pornhub":    pornhub.New(),
		"xvideos":    xvideos.New(),
		"udn":        udn.New(),
		"tiktok":     tiktok.New(),
		"haokan":     haokan.New(),
		"acfun":      acfun.New(),
	}
}
func Domain(u string) (string, error) {
	u = strings.TrimSpace(u)
	var domain string
	bilibiliShortLink := utils.MatchOneOf(u, `^(av|BV|ep)\w+`)
	if len(bilibiliShortLink) > 1 {
		domain = "bilibili"
	} else {
		u, err := url.ParseRequestURI(u)
		if err != nil {
			return "", err
		}
		if u.Host == "haokan.baidu.com" {
			domain = "haokan"
		} else {
			domain = utils.Domain(u.Host)
		}
	}
	return domain, nil
}

// Extract is the main function to extract the data.
func Extract(u string, option types.Options) ([]*types.Data, error) {
	u = strings.TrimSpace(u)
	var domain string

	bilibiliShortLink := utils.MatchOneOf(u, `^(av|BV|ep)\w+`)
	if len(bilibiliShortLink) > 1 {
		bilibiliURL := map[string]string{
			"av": "https://www.bilibili.com/video/",
			"BV": "https://www.bilibili.com/video/",
			"ep": "https://www.bilibili.com/bangumi/play/",
		}
		domain = "bilibili"
		u = bilibiliURL[bilibiliShortLink[1]] + u
	} else {
		u, err := url.ParseRequestURI(u)
		if err != nil {
			return nil, err
		}
		if u.Host == "haokan.baidu.com" {
			domain = "haokan"
		} else {
			domain = utils.Domain(u.Host)
		}
	}
	extractor := extractorMap[domain]
	videos, err := extractor.Extract(u, option)
	if err != nil {
		return nil, err
	}
	for _, v := range videos {
		v.FillUpStreamsData()
	}
	return videos, nil
}
