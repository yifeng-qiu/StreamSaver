// Implements the struct and methods for Progress (individual video download)
// Each download may consists of one or more tracks (video, audio, etc)

package downloader

import "fmt"

// MARK: struct for storing valid video downloads.

type VideoStatus string

const (
	VIDEOSTATUS_NEWVIDEO               VideoStatus = "New Video"
	VIDEOSTATUS_DOWNLOADING            VideoStatus = "Downloading"
	VIDEOSTATUS_MERGING                VideoStatus = "Merging"
	VIDEOSTATUS_REMUXING               VideoStatus = "Remuxing"
	VIDEOSTATUS_WAITING_FOR_CONVERSION VideoStatus = "Conversion pending"
	VIDEOSTATUS_CONVERTING_TO_HLS      VideoStatus = "Converting"
	VIDEOSTATUS_COMPLETED              VideoStatus = "Completed"
	VIDEOSTATUS_ERROR                  VideoStatus = "Error"
	VIDEOSTATUS_PAUSED                 VideoStatus = "Paused"
)

// Struct Video stores information related to one video
type Video struct {
	Index            int              `json:"id"`
	Title            string           `json:"title"`
	Status           VideoStatus      `json:"status"`
	SubStream        []*SubStreamInfo `json:"substreams,omitempty"`
	currentSubstream *SubStreamInfo   `json:"-"`
	substreamCount   int              `json:"-"`
	FileLocation     string           `json:"filelocation"`
	StreamURL        string           `json:"streamurl"`
	Duration         string           `json:"duration"`
	Resolution       string           `json:"resolution"`
}

// Struct SubStreamInfo stores info related to substreams within one video, such as audio, video tracks
type SubStreamInfo struct {
	Index    int     `json:"id"`
	Progress float64 `json:"progress"`
	Size     string  `json:"size"`
	Speed    string  `json:"speed"`
	Eta      string  `json:"eta"`
}

func NewVideo() *Video {
	return &Video{
		Index:          1,
		Title:          "",
		Status:         VIDEOSTATUS_NEWVIDEO,
		SubStream:      make([]*SubStreamInfo, 0),
		substreamCount: 0,
	}
}

func NewSubstream(id int) *SubStreamInfo {
	return &SubStreamInfo{
		Index:    id,
		Progress: 0.0,
		Size:     "",
		Speed:    "",
		Eta:      "",
	}
}

func (v *Video) AddSubstream() {
	v.currentSubstream = NewSubstream(v.substreamCount)
	v.substreamCount += 1
	v.SubStream = append(v.SubStream, v.currentSubstream)
}

func (v *Video) RemoveAllSubstreams() {
	v.substreamCount = 0
	v.SubStream = make([]*SubStreamInfo, 0)
}
func (v *Video) GetProgress(message string) bool {
	if pb, err := ParseProgressBar(message); err == nil {
		v.Index = pb.Playlist_index
		v.Title = pb.Title
		v.currentSubstream.Progress = pb.Progress
		v.currentSubstream.Size = pb.Size
		v.currentSubstream.Speed = pb.Speed
		v.currentSubstream.Eta = pb.Eta
		return true
	} else {
		fmt.Printf("Error when getting progress %s", err.Error())
	}
	return false
}
