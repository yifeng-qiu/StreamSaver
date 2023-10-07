// Implements yt-dlp session. Each session will invoke one instance of yt-dlp and
// download a single video or a single playlist of videos.
package downloader

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yifeng-qiu/StreamSaver/pkg/helper"
)

type State int

const (
	STATE_WAIT State = iota
	STATE_PLAYLIST_TITLE
	STATE_PLAYLIST_SEQ
	STATE_NEW_VIDEO
	STATE_DOWNLOAD_RESUME
	STATE_DOWNLOAD_START
	STATE_DOWNLOAD_IN_PROGRESS
	STATE_DOWNLOAD_COMPLETE
	STATE_MERGE
	STATE_REMUX
	STATE_HLS_CONVERSION
	STATE_SESSION_COMPLETE
	STATE_CANCELED
	STATE_PAUSED
	STATE_ERROR
)

type Status string

const (
	STATUS_WAIT        Status = "wait"
	STATUS_DOWNLOADING Status = "downloading"
	STATUS_COMPLETED   Status = "completed"
	STATUS_CANCELED    Status = "canceled"
	STATUS_PAUSED      Status = "paused"
	STATUS_ERROR       Status = "error"
)

type StdOutContains string

const (
	STDOUT_PLAYLIST_TITLE                StdOutContains = "[download] Downloading playlist: "
	STDOUT_PLAYLIST_SEQ                  StdOutContains = "[download] Downloading item "
	STDOUT_DOWNLOAD_FORMAT               StdOutContains = "Downloading 1 format(s)"
	STDOUT_DOWNLOAD_DESTINATION          StdOutContains = "[download] Destination: "
	STDOUT_DOWNLOAD_PREVIOUSLY_COMPLETED StdOutContains = "has already been downloaded"
	STDOUT_DOWNLOAD_RESUMING             StdOutContains = "[download] Resuming download"
	STDOUT_DOWNLOAD_IN_PROGRESS          StdOutContains = "[progressbar]"
	STDOUT_DOWNLOAD_COMPLETED            StdOutContains = "[download] Download completed"
	STDOUT_WRITE_VIDEO_JSON_METADATA     StdOutContains = "[info] Writing video metadata as JSON to"
	STDOUT_REMUX                         StdOutContains = "[VideoRemuxer]"
	STDOUT_MERGER                        StdOutContains = "[Merger] Merging formats"
	STDOUT_DELETE_PARTS                  StdOutContains = "Deleting original file"
	STDOUT_PLAYLIST_COMPLETE             StdOutContains = "[download] Finished downloading playlist"
)

type Session struct {
	ID             string                        `json:"id"` // the same ID as the SHA key
	StartTime      helper.TimeWithoutNanoseconds `json:"startTime"`
	FinishTime     helper.TimeWithoutNanoseconds `json:"finishTime"`
	URL            string                        `json:"urlraw"`
	state          State                         `json:"-"`
	Status         Status                        `json:"status"`
	Title          string                        `json:"title"` // playlist title or video title depending on the download type
	Playlist_count int                           `json:"playlistCount"`
	Playlist_seq   int                           `json:"playlistIndex"`
	IsPlaylist     bool                          `json:"isPlaylist"`
	Videos         []*Video                      `json:"videos,omitempty"` // one entry for video and multiple for playlist
	currentVideo   *Video                        `json:"-"`
	ffmpegQueue    chan bool                     `json:"-"`
	ffmpegWg       *sync.WaitGroup               `json:"-"`
	sessionPID     int                           `json:"-"`
}

func NewSession(id string, urlstring string, ffmpegQueue chan bool,
	ffmpegWg *sync.WaitGroup) *Session {
	return &Session{
		ID:             id,
		StartTime:      helper.TimeWithoutNanoseconds{Time: time.Now()},
		URL:            urlstring,
		state:          STATE_WAIT,
		Status:         STATUS_WAIT,
		Title:          "",
		Playlist_count: 1,
		Playlist_seq:   1,
		IsPlaylist:     false,
		Videos:         make([]*Video, 0),
		currentVideo:   nil,
		ffmpegQueue:    ffmpegQueue,
		ffmpegWg:       ffmpegWg,
		sessionPID:     -1,
	}
}

func (session *Session) GetSessionStatusString() string {
	return string(session.Status)
}

func (s *Session) Parse(m string) error {
	var err error = nil
	fmt.Printf("DEBUG: yt-dlp returns %v\n", m)
	fmt.Printf("DEBUG: current state %v\n", s.state)
	switch s.state {
	case STATE_WAIT:
		if strings.HasPrefix(m, string(STDOUT_PLAYLIST_TITLE)) {
			_, title, ok := strings.Cut(m, string(STDOUT_PLAYLIST_TITLE))
			if ok {
				s.Title = title
			}
			s.IsPlaylist = true
			s.state = STATE_PLAYLIST_TITLE
		} else if strings.Contains(m, string(STDOUT_DOWNLOAD_FORMAT)) {
			s.addNewVideoToSession()
			s.state = STATE_NEW_VIDEO
			s.IsPlaylist = false
		} else {
			err = errors.New("download did not start")
		}
	case STATE_PLAYLIST_TITLE:
		if strings.Contains(m, string(STDOUT_PLAYLIST_SEQ)) {
			reg, err := regexp.Compile("dwnloading item (\\d+) of (\\d+)")
			if err == nil {
				match := reg.FindStringSubmatch(m)
				if match != nil {
					if val, err := strconv.Atoi(strings.TrimSpace(match[1])); err == nil {
						s.Playlist_seq = val
					}
					if val, err := strconv.Atoi(strings.TrimSpace(match[2])); err == nil {
						s.Playlist_count = val
					}

				} else {
					err = errors.New("failed to extract playlist sequence")
				}
			}
			s.state = STATE_PLAYLIST_SEQ
		} else {
			err = errors.New("did not get message announcing playlist sequence")
		}

	case STATE_PLAYLIST_SEQ:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_FORMAT)) {
			// if the format contains video and audio stream and if the merged
			// file already exists, there will be no progressbar.
			s.addNewVideoToSession()
			s.state = STATE_NEW_VIDEO
		} else {
			err = errors.New("download did not start")
		}

	case STATE_NEW_VIDEO:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_DESTINATION)) ||
			strings.Contains(m, string(STDOUT_DOWNLOAD_PREVIOUSLY_COMPLETED)) {
			s.currentVideo.AddSubstream()
			fmt.Printf("Adding substream %d\n", s.currentVideo.currentSubstream.Index)
			s.state = STATE_DOWNLOAD_START
		} else if strings.Contains(m, string(STDOUT_DOWNLOAD_RESUMING)) {
			s.state = STATE_DOWNLOAD_RESUME
		} else {
			err = errors.New("error when starting to download")
		}
	case STATE_DOWNLOAD_RESUME:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_DESTINATION)) {
			s.currentVideo.AddSubstream()
			fmt.Printf("Adding substream %d\n", s.currentVideo.currentSubstream.Index)
			s.state = STATE_DOWNLOAD_START
		} else {
			err = errors.New("error when starting to download")
		}
	case STATE_DOWNLOAD_START:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_IN_PROGRESS)) {
			s.state = STATE_DOWNLOAD_IN_PROGRESS
			s.currentVideo.GetProgress(m)
			s.currentVideo.Status = VIDEOSTATUS_DOWNLOADING
			if s.IsPlaylist == false {
				s.Title = s.currentVideo.Title
			}
		} else if strings.Contains(m, string(STDOUT_REMUX)) {
			s.extractFileLocation(m)
			s.state = STATE_REMUX
			s.currentVideo.Status = VIDEOSTATUS_REMUXING
		} else {
			err = errors.New("failed to get progress bar")
		}

	case STATE_DOWNLOAD_IN_PROGRESS:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_IN_PROGRESS)) {
			s.currentVideo.GetProgress(m)
		} else if strings.Contains(m, string(STDOUT_DOWNLOAD_COMPLETED)) {
			s.state = STATE_DOWNLOAD_COMPLETE
		} else {
			err = errors.New("did not complete download")
		}
	case STATE_DOWNLOAD_COMPLETE:
		if strings.Contains(m, string(STDOUT_DOWNLOAD_DESTINATION)) ||
			strings.Contains(m, string(STDOUT_DOWNLOAD_PREVIOUSLY_COMPLETED)) {
			s.state = STATE_DOWNLOAD_START
			s.currentVideo.AddSubstream()
		} else if strings.Contains(m, string(STDOUT_MERGER)) {
			s.state = STATE_MERGE
			s.currentVideo.Status = VIDEOSTATUS_MERGING

		} else if strings.Contains(m, string(STDOUT_REMUX)) {
			s.extractFileLocation(m)
			s.state = STATE_REMUX
			s.currentVideo.Status = VIDEOSTATUS_REMUXING
		} else {
			err = errors.New("download completed but did not move to the next step")
		}

	case STATE_MERGE:
		if strings.Contains(m, string(STDOUT_REMUX)) {
			s.extractFileLocation(m)
			s.state = STATE_REMUX
			s.currentVideo.Status = VIDEOSTATUS_REMUXING
		} else {
			err = errors.New("did not receive VideoRemuxer message")
		}
	case STATE_REMUX:
		/*
			While in this state, we can expect 3 situations
			1. about to download the next video in a playlist
			2. the playlist is completed
			3. yt-dlp exits without error
			In all cases, the current video struct is handed over to HLS conversion process
		*/
		if strings.Contains(m, string(STDOUT_PLAYLIST_SEQ)) {
			s.state = STATE_PLAYLIST_SEQ
			s.GetFileSpecs()
			// Start HLS conversion on the last video
			err := s.SetupHLSConversion()
			if err != nil {
				err = fmt.Errorf("error when starting HLS conversion %w", err)
			}
		} else if s.IsPlaylist && strings.Contains(m, string(STDOUT_PLAYLIST_COMPLETE)) {
			s.GetFileSpecs()
			s.state = STATE_HLS_CONVERSION
			err := s.SetupHLSConversion()
			if err != nil {
				err = fmt.Errorf("error when starting HLS conversion %w", err)
			}
		} else {
			err = errors.New("extraneous stdout after Remux")
		}
	default:
		err = errors.New("illegal state")
	}
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		s.state = STATE_ERROR
	}
	s.updateStatus()

	return err

}

func (s *Session) extractFileLocation(m string) {
	if strings.Contains(m, "Not remuxing") {
		reg := regexp.MustCompile("\"(.+)\"")
		if match := reg.FindStringSubmatch(m); match != nil {
			s.currentVideo.FileLocation = match[1]
		}

	} else if strings.Contains(m, "Remuxing video") {
		reg := regexp.MustCompile("Destination: (.+)")
		if match := reg.FindStringSubmatch(m); match != nil {
			s.currentVideo.FileLocation = match[1]
		}
	}
}

func (s *Session) updateStatus() {
	switch s.state {
	case STATE_WAIT:
		s.Status = STATUS_WAIT
	case STATE_PLAYLIST_TITLE,
		STATE_PLAYLIST_SEQ,
		STATE_NEW_VIDEO,
		STATE_DOWNLOAD_START,
		STATE_DOWNLOAD_IN_PROGRESS,
		STATE_DOWNLOAD_COMPLETE,
		STATE_MERGE,
		STATE_REMUX,
		STATE_HLS_CONVERSION:
		s.Status = STATUS_DOWNLOADING
	case STATE_CANCELED:
		s.Status = STATUS_CANCELED
	case STATE_SESSION_COMPLETE:
		s.Status = STATUS_COMPLETED
	case STATE_PAUSED:
		s.Status = STATUS_PAUSED
	case STATE_ERROR:
		s.Status = STATUS_ERROR
	}
}

func (s *Session) addNewVideoToSession() {
	s.currentVideo = NewVideo()
	s.Videos = append(s.Videos, s.currentVideo)

}

// StartHLSConversion invokes ffmpeg to convert any video into the hls format suitable for streaming
func StartHLSConversion(input string, output string, video *Video, pid *int) error {
	defer func() { *pid = -1 }()
	if video == nil {
		fmt.Println("The video does not exist")
		return errors.New("the video does not exist")
	}
	cmd := exec.Command("ffmpeg", "-i", input, "-start_number", "0",
		"-hls_time", "10", "-hls_list_size", "0", "-f", "hls", output, "-loglevel", "error")
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error when starting ffmpeg %w", err)
	} else {
		*pid = cmd.Process.Pid
	}

	err = cmd.Wait()

	if err != nil {
		video.Status = VIDEOSTATUS_ERROR
		fmt.Printf("error during HLS conversio %s", err.Error())
		return fmt.Errorf("error during HLS conversio %w", err)
	} else {
		fmt.Printf("Conversion to HLS completed, stored at %s\n", output)
		unescapedPath := strings.TrimPrefix(output, "/media/download")
		pathComponents := strings.Split(unescapedPath, "/")

		for i, component := range pathComponents {
			pathComponents[i] = url.PathEscape(component)
		}
		encodedPath := strings.Join(pathComponents, "/")
		video.StreamURL = encodedPath
		video.Status = VIDEOSTATUS_COMPLETED

		return nil
	}
}

// SetupHLSConversion prepares for ffmpeg conversion
func (s *Session) SetupHLSConversion() error {
	s.currentVideo.Status = VIDEOSTATUS_WAITING_FOR_CONVERSION
	filename := filepath.Base(s.currentVideo.FileLocation)
	hlsPath := strings.TrimRight(helper.SHAFromString(filename), "=")

	newFolder := filepath.Join("/media/hls", hlsPath)
	os.Mkdir(newFolder, 0755)
	hlsFilename := filepath.Join(newFolder, hlsPath+".m3u8")

	fmt.Printf("HLS files will be saved to %s\n", hlsFilename)
	s.currentVideo.Status = VIDEOSTATUS_WAITING_FOR_CONVERSION

	fmt.Printf("ffmpeg conversion scheduled for %s\n", filename)
	s.ffmpegWg.Add(1)
	go func(source string, target string, video *Video) {
		defer s.ffmpegWg.Done()
		s.ffmpegQueue <- true
		fmt.Printf("Starting ffmpeg conversion\n")
		video.Status = VIDEOSTATUS_CONVERTING_TO_HLS
		StartHLSConversion(source, target, video, &s.sessionPID)
		<-s.ffmpegQueue
	}(s.currentVideo.FileLocation, hlsFilename, s.currentVideo)
	return nil
}

// Extract playback duration using ffprobe
func GetMediaPlaybackDuration(input string) string {
	durationString := ""
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", input)
	duration, err := cmd.Output()
	if err != nil {
		fmt.Printf("Unable to obtain the duration of the file, %s", err.Error())
	} else {

		duration_parts := strings.Split(string(duration), ".")
		if len(duration_parts) >= 1 {
			if seconds, err := strconv.Atoi(duration_parts[0]); err == nil {
				duration := time.Duration(seconds) * time.Second
				hours := int(duration.Hours())
				minutes := int(duration.Minutes()) % 60
				seconds := int(duration.Seconds()) % 60

				durationString += fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
			}
		}
		if len(duration_parts) >= 2 {
			if milliseconds, err := strconv.Atoi(duration_parts[1]); err == nil {
				durationString += fmt.Sprintf(":%d", milliseconds)
			}
		}

	}
	return durationString
}

// Get media resolution using ffprobe
func GetMediaResolution(input string) string {

	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", input)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("Unable to obtain the resolution of the file, %s", err.Error())
		return ""
	} else {
		format := string(stdout)
		format = strings.TrimRight(format, "\n\r ")
		return format
	}
}

func (s *Session) GetFileSpecs() {
	s.currentVideo.Duration = GetMediaPlaybackDuration(s.currentVideo.FileLocation)
	s.currentVideo.Resolution = GetMediaResolution(s.currentVideo.FileLocation)
}
