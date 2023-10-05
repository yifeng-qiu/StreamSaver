package downloader

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/yifeng-qiu/ytdlp_backend/pkg/helper"
)

type Downloader struct {
	shaKey          string
	urlstring       string
	sessionPID      int
	currentSession  *Session
	downloadQueue   chan bool
	ffmpegQueue     chan bool
	ffmpeg_wg       sync.WaitGroup
	postSessionFunc postSession
}

func (d *Downloader) status() string {
	if d.currentSession != nil {
		return d.currentSession.GetSessionStatusString()
	} else {
		return ""
	}
}

func (d *Downloader) Terminate() bool {
	fmt.Printf("DEBUG: received request to cancel job:%s \n", d.shaKey)
	fmt.Printf("DEBUG: the session PID is :%d \n", d.sessionPID)
	ret := false
	if d.sessionPID != -1 && d.sessionPID != 0 {
		cmd := exec.Command("kill", fmt.Sprintf("%d", d.sessionPID))
		err := cmd.Run()
		if err == nil {
			ret = true
		}
	}

	if d.currentSession.sessionPID != -1 && d.currentSession.sessionPID != 0 {
		cmd := exec.Command("kill", fmt.Sprintf("%d", d.currentSession.sessionPID))
		err := cmd.Run()
		if err == nil {
			ret = true
		}

	}
	return ret
}

func (d *Downloader) Start() {
	go func() {
		d.ffmpeg_wg = sync.WaitGroup{}
		d.downloadQueue <- true
		d.ytdlp()
		<-d.downloadQueue
		fmt.Println("DEBUG: ytdlp execution completed")
		d.ffmpeg_wg.Wait()
		if d.currentSession.state == STATE_HLS_CONVERSION {
			d.currentSession.FinishTime = helper.TimeWithoutNanoseconds{Time: time.Now()}
			d.currentSession.state = STATE_SESSION_COMPLETE
			d.currentSession.Status = STATUS_COMPLETED
		} else {
			d.currentSession.state = STATE_ERROR
			d.currentSession.Status = STATUS_ERROR
		}

	}()
}

func (d *Downloader) ytdlp() {
	defer func() { d.sessionPID = 0 }()

	cmd := exec.Command("yt-dlp", d.urlstring)
	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Debug: Error opening cmd.StdoutPipe(): %v", err.Error())
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Debug: error when trying to run yt-dlp command: %v", err.Error())
	} else {
		d.sessionPID = cmd.Process.Pid
		fmt.Printf("Debug: the PID of the yt-dlp session is %d\n", d.sessionPID)
	}

	scannerStdout := bufio.NewScanner(stdout)
	// scannerStderr := bufio.NewScanner(stderr)

	combinedOutput := make(chan string, 10) // channel combining both Stdout and Stderr as well as Cmds originating from the server

	wg.Add(1)
	go func() {
		defer wg.Done()
		for scannerStdout.Scan() {
			m := scannerStdout.Text()
			if strings.TrimSpace(m) != "" {
				combinedOutput <- m
				// fmt.Println("scannerStdout: ", m)
			}
		}
	}()
	go func() {
		wg.Wait()
		close(combinedOutput)
	}()

	if d.currentSession == nil {
		d.currentSession = NewSession(d.shaKey, d.urlstring, d.ffmpegQueue, &d.ffmpeg_wg)
		d.postSessionFunc(d.currentSession)
	}

	// This is the main loop of the yt-dlp session.

	for m := range combinedOutput {
		if hasValidPrefix(m) {
			err = d.currentSession.Parse(m)
			if err != nil {
				if d.currentSession.currentVideo != nil {
					d.currentSession.currentVideo.Status = VIDEOSTATUS_ERROR
				}
				fmt.Printf("Error when parsing output due to %s\n", err.Error())
				fmt.Printf("Terminating download ...\n")
				d.Terminate()
			}
		}
	}
	err = cmd.Wait()
	if err != nil {
		// Conditions on which this err will be triggered:
		// 1. if the process is terminated by kill, it will result in error
		// 2. unsupported url returned by yt-dlp itself.
		fmt.Printf("Debug: error after cmd.Wait caused by %v \n", err.Error())
		if strings.Contains(err.Error(), "terminated") {
			d.currentSession.state = STATE_CANCELED
		} else {
			d.currentSession.state = STATE_ERROR
			fmt.Printf("DEBUG: irrecoverable error occurred during download %s", err.Error())
		}

	} else {
		// At this point the Session.State should be STATE_REMUX
		fmt.Printf("yt-dlp finished without error, current state is %d\n", d.currentSession.state)
		if d.currentSession.state == STATE_REMUX {
			fmt.Println("Getting file spec")
			d.currentSession.GetFileSpecs()
			d.currentSession.SetupHLSConversion()
			d.currentSession.state = STATE_HLS_CONVERSION
		}
	}
}

func validPrefixes() []string {
	return []string{"[download]", "[info]", "[progressbar]", "[Merger]", "[VideoRemuxer]"}
}

func hasValidPrefix(s string) bool {
	for _, prefix := range validPrefixes() {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
