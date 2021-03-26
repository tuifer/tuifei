package youtube

import (
	"errors"
	"fmt"
	ytdl "github.com/kkdai/youtube/v2"
	"github.com/tuifer/tuifei/extractors/types"
	"github.com/tuifer/tuifei/request"
	"github.com/tuifer/tuifei/utils"
	"strconv"
	"strings"
)

type streamFormat struct {
	Itag          int    `json:"itag"`
	URL           string `json:"url"`
	MimeType      string `json:"mimeType"`
	ContentLength string `json:"contentLength"`
	QualityLabel  string `json:"qualityLabel"`
	AudioQuality  string `json:"audioQuality"`
}

type formats []*streamFormat

func (playerAdaptiveFormats formats) filterPlayerAdaptiveFormats(videoInfoFormats ytdl.FormatList) (filter formats) {
	videoInfoFormatMap := make(map[int]struct{}, len(videoInfoFormats))
	for _, f := range videoInfoFormats {
		videoInfoFormatMap[f.ItagNo] = struct{}{}
	}
	for _, f := range playerAdaptiveFormats {
		if _, ok := videoInfoFormatMap[f.Itag]; ok {
			filter = append(filter, f)
		}
	}
	return
}

const referer = "https://www.youtube.com"

type extractor struct{}

// New returns a youtube extractor.
func New() types.Extractor {
	return &extractor{}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(uri string, option types.Options) ([]*types.Data, error) {
	var err error
	if !option.Playlist {
		return []*types.Data{youtubeDownload(uri)}, nil
	}
	listIDs := utils.MatchOneOf(uri, `(list|p)=([^/&]+)`)
	if listIDs == nil || len(listIDs) < 3 {
		return nil, types.ErrURLParseFailed
	}
	listID := listIDs[2]
	if len(listID) == 0 {
		return nil, errors.New("can't get list ID from URL")
	}

	html, err := request.Get("https://www.youtube.com/playlist?list="+listID, referer, nil)
	if err != nil {
		return nil, err
	}
	// "videoId":"OQxX8zgyzuM","thumbnail"
	videoIDs := utils.MatchAll(html, `"videoId":"([^,]+?)","thumbnail"`)
	needDownloadItems := utils.NeedDownloadList(option.Items, option.ItemStart, option.ItemEnd, len(videoIDs))
	extractedData := make([]*types.Data, len(needDownloadItems))
	wgp := utils.NewWaitGroupPool(option.ThreadNumber)
	dataIndex := 0
	for index, videoID := range videoIDs {
		if !utils.ItemInSlice(index+1, needDownloadItems) || len(videoID) < 2 {
			continue
		}
		u := fmt.Sprintf(
			"https://www.youtube.com/watch?v=%s&list=%s", videoID[1], listID,
		)
		wgp.Add()
		go func(index int, u string, extractedData []*types.Data) {
			defer wgp.Done()
			extractedData[index] = youtubeDownload(u)
		}(dataIndex, u, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

// youtubeDownload download function for single url
func youtubeDownload(uri string) *types.Data {
	vid := utils.MatchOneOf(
		uri,
		`watch\?v=([^/&]+)`,
		`youtu\.be/([^?/]+)`,
		`embed/([^/?]+)`,
		`v/([^/?]+)`,
	)
	if vid == nil || len(vid) < 2 {
		return types.EmptyData(uri, errors.New("can't find vid"))
	}

	videoURL := fmt.Sprintf(
		"https://www.youtube.com/watch?v=%s",
		vid[1],
	)
	client := ytdl.Client{}
	videoInfo, err := client.GetVideo(videoURL)
	if err != nil {
		return types.EmptyData(uri, err)
	}
	title := videoInfo.Title
	streams, err := extractVideoURL(videoInfo)
	if err != nil {
		return types.EmptyData(uri, err)
	}

	return &types.Data{
		Site:    "YouTube youtube.com",
		Title:   title,
		VideoId: fmt.Sprintf("%s", vid[1]),
		Type:    "video",
		Streams: streams,
		URL:     uri,
	}
}

func getStreamExt(streamType string) string {
	// video/webm; codecs="vp8.0, vorbis" --> webm
	exts := utils.MatchOneOf(streamType, `(\w+)/(\w+);`)
	if exts == nil || len(exts) < 3 {
		return ""
	}
	return exts[2]
}
func getRealURL(videoFormat *ytdl.Format, ext string) (*types.Part, error) {
	size, _ := strconv.ParseInt(videoFormat.ContentLength, 10, 64)
	return &types.Part{
		URL:  videoFormat.URL,
		Size: size,
		Ext:  ext,
	}, nil
}
func genStream(videoFormat *ytdl.Format) (*types.Stream, error) {
	streamType := videoFormat.MimeType
	ext := getStreamExt(streamType)

	if ext == "" {
		return nil, fmt.Errorf("unable to get file extension of MimeType %s", streamType)
	}

	video, err := getRealURL(videoFormat, ext)
	if err != nil {
		return nil, err
	}

	var quality string
	if videoFormat.QualityLabel != "" {
		quality = fmt.Sprintf("%s %s", videoFormat.QualityLabel, streamType)
	} else {
		quality = streamType
	}

	return &types.Stream{
		ID:      strconv.Itoa(videoFormat.ItagNo),
		Parts:   []*types.Part{video},
		Quality: quality,
		NeedMux: true,
	}, nil
}
func extractVideoURL(videoInfo *ytdl.Video) (map[string]*types.Stream, error) {
	streams := make(map[string]*types.Stream, len(videoInfo.Formats))

	for _, f := range videoInfo.Formats {

		stream, err := genStream(&f)
		if err != nil {
			return nil, err
		}

		streams[strconv.Itoa(f.ItagNo)] = stream
	}
	var fM4aMedium, fM4aLow, fWebmMedium, fWebmLow *ytdl.Format
	for _, f := range videoInfo.Formats {
		switch {
		case strings.HasPrefix(f.MimeType, "audio/mp4"):
			if f.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
				fM4aMedium = &f
			} else {
				fM4aLow = &f
			}
		case strings.HasPrefix(f.MimeType, "audio/webm"):

			if f.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
				fWebmMedium = &f
			} else {
				fWebmLow = &f
			}
		}

		if fM4aMedium != nil && fWebmMedium != nil {
			break
		}
	}
	var audioWebm *types.Part
	if fWebmMedium != nil {
		audioURL, err := getRealURL(fWebmMedium, "webm")
		if err != nil {
			return nil, err
		}
		audioWebm = audioURL
	} else if fWebmLow != nil {
		audioURL, err := getRealURL(fWebmLow, "webm")
		if err != nil {
			return nil, err
		}
		audioWebm = audioURL
	}
	var audioM4a *types.Part
	if fM4aMedium != nil {
		audioURL, err := getRealURL(fM4aMedium, "m4a")
		if err != nil {
			return nil, err
		}
		audioM4a = audioURL
	} else if fM4aLow != nil {
		audioURL, err := getRealURL(fM4aLow, "m4a")
		if err != nil {
			return nil, err
		}
		audioM4a = audioURL
	}
	for _, f := range videoInfo.Formats {

		stream, err := genStream(&f)
		if err != nil {
			return nil, err
		}

		// append audio stream only for adaptive video streams (not audio)
		switch {
		case strings.HasPrefix(f.MimeType, "video/mp4"):
			if audioM4a != nil {
				stream.Parts = append(stream.Parts, audioM4a)
				stream.Quality = fmt.Sprintf("%s m4a音频分轨", stream.Quality)
			}
		case strings.HasPrefix(f.MimeType, "video/webm"):
			if audioWebm != nil {
				stream.Parts = append(stream.Parts, audioWebm)
				stream.Quality = fmt.Sprintf("%s webm音频分轨", stream.Quality)
			}
		}

		streams[strconv.Itoa(f.ItagNo)] = stream
	}
	return streams, nil
}
