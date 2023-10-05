// This package implements a simple CRUD HTTP server
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/yifeng-qiu/ytdlp_backend/pkg/downloader"
	"github.com/yifeng-qiu/ytdlp_backend/pkg/helper"
)

type RequestHandler struct {
	Requests        map[string]Request
	DownloadManager downloader.DownloadManager
}

type Request struct {
	URL         string    // URL string
	ReceiveTime time.Time // timestamp when the request was received
	Status      string
}

var ErrIDNotFound = fmt.Errorf("ID not found")
var ErrEmptyRequestText = fmt.Errorf("requested URL cannot be empty")
var ErrURLAlreadyExisted = fmt.Errorf("requested URL already existed")
var ErrUnsupportedURL = fmt.Errorf("the provided URL is not supported by YT-DLP")

func (s *RequestHandler) Insert(urlstring string) (string, error) {
	fmt.Println("the received URL is: ", urlstring)
	if urlstring != "" {
		sha := helper.SHAFromString(urlstring)
		_, ok := s.Requests[sha] // check if the request has been posted before
		if !ok {
			newRequest := Request{
				ReceiveTime: time.Now(),
				URL:         urlstring,
			}
			s.Requests[sha] = newRequest
			return sha, nil
		} else {
			return "", ErrURLAlreadyExisted
		}
	} else {
		return "", ErrEmptyRequestText
	}
}

func (s *RequestHandler) Retrieve(sha string) (*Request, error) {
	request, ok := s.Requests[sha]
	if !ok {
		return nil, ErrIDNotFound
	} else {
		return &request, nil
	}
}

func (s *RequestHandler) NewHTTPServer(addr string) *http.Server {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 || parts[0] == "" || (parts[0] != "localhost" && parts[0] != "127.0.0.1") {
		fmt.Println("Binding to all addresses unknown to this interface!")
	}
	mux := http.NewServeMux()
	NewServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	mux.Handle("/", s)
	return NewServer
}

func (s *RequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// fmt.Println(req.Method)

	switch req.Method {
	case "GET":
		idQuery := req.URL.Query()
		if len(idQuery) == 0 {
			fmt.Fprintf(w, "")
		} else if idQuery.Has("id") {
			arg := idQuery.Get("id")
			if arg == "" {
				err := json.NewEncoder(w).Encode(s.DownloadManager.SessionsInfo)
				if err != nil {
					fmt.Println(err.Error())
				}
			} else {
				request, err := s.Retrieve(arg)
				if err != nil {
					fmt.Fprintf(w, err.Error())
				} else {
					json.NewEncoder(w).Encode(request)
				}
			}
		}

	case "POST":
		myURL := req.FormValue("url")
		decodedValue, err := url.QueryUnescape(myURL)
		if err == nil {
			fmt.Println("Received POST request with text: ", decodedValue)
		}
		if myURL != "" {
			if newSHA, err := s.Insert(myURL); err != nil {
				fmt.Println(err)
				// s.DownloadManager.NewDownload(newSHA, myURL)
			} else {
				fmt.Fprintf(w, "A new request for (%s) has been registered and the SHA of the link is %s", myURL, newSHA)
				fmt.Fprintf(os.Stdout, "A new request for (%s) has been registered and the SHA of the link is %s\n", myURL, newSHA)
				fmt.Printf("Total activities so far: %d\n", len(s.Requests))
				s.DownloadManager.NewDownload(newSHA, myURL)
			}
		} else {
			fmt.Fprintf(w, "request cannot be empty")
		}
	case "DELETE":
		shaKey := strings.TrimLeft(req.URL.Path, "/")
		fmt.Println("DEBUG: Delete requested shakey is ", shaKey)
		if s.DownloadManager.CancelDownload(shaKey) {

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Job deleted successfully.\n")
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Error when trying to cancel the download job.\n")
		}
		delete(s.Requests, shaKey)
	default:
		fmt.Fprintf(w, "Invalid HTTP method! %s\n", req.Method)
	}

}
