package downloader

import (
	"fmt"
	"net/url"
)

type DownloadManager struct {
	Downloaders    map[string]*Downloader
	DownloadQueues map[string]chan bool // queue per domain for scheduling download sessions
	SessionsInfo   []*Session
	ffmpegQueue    chan bool // only allow one instance of ffmpeg
}

func NewDownloadManager() DownloadManager {
	return DownloadManager{
		Downloaders:    make(map[string]*Downloader),
		DownloadQueues: make(map[string]chan bool),
		SessionsInfo:   make([]*Session, 0),
		ffmpegQueue:    make(chan bool, 1),
	}
}

var sessionLock chan int = make(chan int)
var downloaderLock chan int = make(chan int)

type postSession func(session *Session)

func (dm *DownloadManager) PostSession(session *Session) {
	dm.SessionsInfo = append(dm.SessionsInfo, session)

}

func (dm *DownloadManager) removeSession(shaKey string) {
	var idx int = -1
	for idx = range dm.SessionsInfo {
		if dm.SessionsInfo[idx].ID == shaKey {
			break
		}
	}
	if idx != -1 {
		dm.SessionsInfo = append(dm.SessionsInfo[:idx], dm.SessionsInfo[idx+1:]...)
	}
}

func (dm *DownloadManager) removeDownloader(shaKey string) {
	delete(dm.Downloaders, shaKey)
}

func (dm *DownloadManager) NewDownload(shaKey string, urlstring string) {
	downloader, ok := dm.Downloaders[shaKey]
	if ok {
		// existing downloader, restart the download
		downloader.Start()
	} else {
		newURL, err := url.Parse(urlstring)
		if err == nil {
			host := newURL.Host
			queue, ok := dm.DownloadQueues[host]
			if !ok {
				queue = make(chan bool, 2)
				dm.DownloadQueues[host] = queue

			}
			newDownloader := &Downloader{
				shaKey:          shaKey,
				urlstring:       urlstring,
				downloadQueue:   queue,
				ffmpegQueue:     dm.ffmpegQueue,
				postSessionFunc: dm.PostSession,
			}

			dm.Downloaders[shaKey] = newDownloader

			newDownloader.Start()

		} else {
			fmt.Print(err.Error())
		}
	}
}

func (dm *DownloadManager) FindDownloader(shaKey string) *Downloader {
	if downloader, ok := dm.Downloaders[shaKey]; ok {
		return downloader

	} else {
		return nil
	}
}

func (dm *DownloadManager) CancelDownload(shaKey string) bool {
	// if the downloader is present and its sessionPID is known issue command and kill it
	downloader := dm.FindDownloader(shaKey)
	if downloader != nil {
		downloader.Terminate()
		// remove associated session from sessionsInfo
		dm.removeSession(shaKey)
		// remove downloader from the downloader map
		dm.removeDownloader(shaKey)
		return true
	}
	return false
}
