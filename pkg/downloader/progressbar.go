// Parser functions for dealing with progress outputs from yt-dlp.
// The format is
// Playlist$$__$$Playlist_index$$__$$Playlist_count$$__$$Title$$__$$Combined string of Progress, Size, Speed and Eta
package downloader

import (
	"errors"
	"regexp"
	"strconv"
)

const (
	progress = `([\d\.]+)%`
	size     = `([\d\.]+[KMGT]iB)`
	speed    = `([\d\.]+[KMGT]iB/s)`
	eta      = `((?:\d{2}:){1,2}\d{2})`
)

type ProgressBar struct {
	Playlist       string  `json:"playlist"`
	Playlist_index int     `json:"id"`
	Playlist_count int     `json:"playlistCount"`
	Title          string  `json:"title"`
	Progress       float64 `json:"progress"`
	Size           string  `json:"size"`
	Speed          string  `json:"speed"`
	Eta            string  `json:"eta"`
}

var ProgressBarRegExps = map[string]*regexp.Regexp{
	"progress": regexp.MustCompile(progress),
	"size":     regexp.MustCompile(size),
	"speed":    regexp.MustCompile(speed),
	"eta":      regexp.MustCompile(eta),
}

// ParseProgressBar parses the progressbar outputs of yt-dlp and store parsed info in a ProgressBar stuct.
// - ProgressBar: pointer to a ProgressBar struct
func ParseProgressBar(from string) (*ProgressBar, error) {
	p := ProgressBar{
		Playlist:       "",
		Playlist_index: 1,
		Playlist_count: 1,
		Title:          "",
		Progress:       0.0,
		Size:           "",
		Speed:          "",
		Eta:            "",
	}
	// splitting fields by $$_$$ sign
	x := regexp.MustCompile(`\$\$__\$\$`).Split(from, -1)
	// the returned slice of strings must have a length of 5
	if len(x) < 5 {
		return nil, errors.New("invalid number of fields") // the string cannot be parsed correctly
	}
	p.Playlist = x[0]
	val, err := strconv.Atoi(x[1])
	if err == nil {
		p.Playlist_index = val
	} else {
		p.Playlist_index = 1
	}

	val, err = strconv.Atoi(x[2])
	if err == nil {
		p.Playlist_count = val
	} else {
		p.Playlist_count = 1
	}

	p.Title = x[3]
	if len(p.Title) == 0 {
		return nil, errors.New("could not extract title")
	}

	// The remainder of the fields will not throw an error even if not parsed correctly
	remainder := x[4]
	str := findMatch(ProgressBarRegExps["progress"], remainder)
	if len(str) != 0 {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			p.Progress = f / 100.0
		}
	}
	p.Size = findMatch(ProgressBarRegExps["size"], remainder)

	p.Speed = findMatch(ProgressBarRegExps["speed"], remainder)

	p.Eta = findMatch(ProgressBarRegExps["eta"], remainder)

	return &p, nil
}

// Helper function for copying info between two ProgressBar structs
func (p *ProgressBar) CopyInto(into *ProgressBar) {

	into.Playlist = p.Playlist
	into.Playlist_count = p.Playlist_count
	into.Playlist_index = p.Playlist_index
	into.Title = p.Title
	into.Progress = p.Progress
	into.Size = p.Size
	into.Speed = p.Speed
	into.Eta = p.Eta
}

// Helper function for extracting information from a progressbar string.
func findMatch(reg *regexp.Regexp, from string) string {
	match := reg.FindSubmatch([]byte(from))
	if match != nil {
		return string(match[1])
	} else {
		return ""
	}
}
