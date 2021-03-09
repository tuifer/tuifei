package downloader

import (
	"testing"

	"github.com/tuifer/tuifei/extractors/types"
)

func TestDownload(t *testing.T) {
	testCases := []struct {
		name string
		data *types.Data
	}{
		{
			name: "normal test",
			data: &types.Data{
				Site:  "douyin",
				Title: "test",
				Type:  types.DataTypeVideo,
				URL:   "https://www.douyin.com",
				Streams: map[string]*types.Stream{
					"default": {
						ID: "default",
						Parts: []*types.Part{
							{
								URL:  "https://aweme.snssdk.com/aweme/v1/playwm/?video_id=v0200f9a0000bc117isuatl67cees890&line=0",
								Size: 4927877,
								Ext:  "mp4",
							},
						},
					},
				},
			},
		},
		{
			name: "multi-stream test",
			data: &types.Data{
				Site:  "douyin",
				Title: "test2",
				Type:  types.DataTypeVideo,
				URL:   "https://www.douyin.com",
				Streams: map[string]*types.Stream{
					"miaopai": {
						ID: "miaopai",
						Parts: []*types.Part{
							{
								URL:  "https://txycdn.miaopai.com/stream/KwR26jUGh2ySnVjYbQiFmomNjP14LtMU3vi6sQ__.mp4?ssig=6594aa01a78e78f50c65c164d186ba9e&time_stamp=1537070910786",
								Size: 4011590,
								Ext:  "mp4",
							},
						},
						Size: 4011590,
					},
					"douyin": {
						ID: "douyin",
						Parts: []*types.Part{
							{
								URL:  "https://aweme.snssdk.com/aweme/v1/playwm/?video_id=v0200f9a0000bc117isuatl67cees890&line=0",
								Size: 4927877,
								Ext:  "mp4",
							},
						},
						Size: 4927877,
					},
				},
			},
		},
		
	}
	for _, testCase := range testCases {
		err := New(Options{}).Download(testCase.data)
		if err != nil {
			t.Error(err)
		}
	}
}
