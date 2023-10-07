// This package implements a simple CRUD HTTP server
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/yifeng-qiu/ytdlp_backend/pkg/downloader"
	"github.com/yifeng-qiu/ytdlp_backend/pkg/helper"
)

// Type RequestHandler associates a dictionary of requests with a DownloadManager
type RequestHandler struct {
	Requests        map[string]Request
	DownloadManager downloader.DownloadManager
}

// Type Request represents a received URL request
type Request struct {
	URL         string    // URL string
	ReceiveTime time.Time // timestamp when the request was received
	Status      string
}

type NewURLResponse struct {
	URL            string `json:"url"`
	ShaKey         string `json:"shaKey"`
	TotalDownloads int    `json:"totalDownloads"`
}

var ErrIDNotFound = fmt.Errorf("ID not found")
var ErrEmptyRequestText = fmt.Errorf("requested URL cannot be empty")
var ErrURLAlreadyExisted = fmt.Errorf("requested URL already existed")
var ErrUnsupportedURL = fmt.Errorf("the provided URL is not supported by YT-DLP")

// Insert adds new URL request to the queue
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

// Retrieve a Request from provided SHA hash
func (s *RequestHandler) Retrieve(sha string) (*Request, error) {
	request, ok := s.Requests[sha]
	if !ok {
		return nil, ErrIDNotFound
	} else {
		return &request, nil
	}
}

func HealthCheckHandler(w http.ResponseWriter, req *http.Request) {
	WriteJSONMessage(w, `{"alive": true}`)
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// io.WriteString(w, `{"alive": true}`)
}

func (s *RequestHandler) GetAllDownloads(w http.ResponseWriter, req *http.Request) {
	WriteJSONMessage(w, s.DownloadManager.SessionsInfo)
}

func (s *RequestHandler) HandleSingleDownload(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	shaKey := vars["id"]
	request, err := s.Retrieve(shaKey)
	if err != nil {
		// ID does not exist
		WriteHttpErrorMessage(w, shaKey+" does not exist", http.StatusNotFound)
	} else {
		switch req.Method {
		case "GET":
			WriteJSONMessage(w, request)
		case "UPDATE":
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
		case "DELETE":
			if s.DownloadManager.CancelDownload(shaKey) {
				WriteJSONMessage(w, `{"deletion": true}`)
				// w.Header().Set("Content-Type", "application/json")
				// w.WriteHeader(http.StatusOK)
				// io.WriteString(w, `{"deletion": true}`)
			} else {
				WriteJSONMessage(w, `{"deletion": false}`)
				// w.Header().Set("Content-Type", "application/json")
				// w.WriteHeader(http.StatusInternalServerError)
				// io.WriteString(w, `{"deletion": false}`)
			}
			delete(s.Requests, shaKey)
		default:
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
		}
	}
}

// Handles new URL request sent with POST method. The server expects the URL to be provided
// as FORM data
func (s *RequestHandler) NewURLHandler(w http.ResponseWriter, req *http.Request) {
	myURL := req.FormValue("url")
	decodedValue, err := url.QueryUnescape(myURL)
	if err == nil {
		fmt.Println("Received POST request with text: ", decodedValue)
	}
	if myURL == "" {
		WriteHttpErrorMessage(w, "request cannot be empty", http.StatusBadRequest)
	} else {
		if newSHA, err := s.Insert(myURL); err != nil {
			WriteHttpErrorMessage(w, "unable to create a new request", http.StatusInternalServerError)
		} else {
			newResponse := NewURLResponse{
				URL:            myURL,
				ShaKey:         newSHA,
				TotalDownloads: len(s.Requests),
			}
			WriteJSONMessage(w, newResponse)
			fmt.Fprintf(os.Stdout, "A new request for (%s) has been registered and the SHA of the link is %s\n", myURL, newSHA)
			fmt.Printf("Total activities so far: %d\n", len(s.Requests))
			s.DownloadManager.NewDownload(newSHA, myURL)

		}
	}
}

// Helper function for writting an error message to HTTP response
func WriteHttpErrorMessage(w http.ResponseWriter, errText string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	io.WriteString(w, errText)
}

// Helper function for encoding a response in JSON and set the proper header
func WriteJSONMessage(w http.ResponseWriter, v any) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if err := encoder.Encode(v); err != nil {
		// Handle the error, e.g., send a 500 Internal Server Error response.
		WriteHttpErrorMessage(w, "Failed to encode JSON data", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buffer.Bytes())
	}
}

// Creates a new HTTP Server
func (s *RequestHandler) NewHTTPServer(addr string) *http.Server {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 || parts[0] == "" || (parts[0] != "localhost" && parts[0] != "127.0.0.1") {
		fmt.Println("Will bind to all addresses!")
	}
	r := mux.NewRouter()
	r.HandleFunc("/new", s.NewURLHandler).Methods("POST")
	r.HandleFunc("/urls", s.GetAllDownloads).Methods("GET")
	r.HandleFunc("/urls/{id}", s.HandleSingleDownload).Methods("GET", "UPDATE", "DELETE")
	r.HandleFunc("/", HealthCheckHandler).Methods("GET")

	NewServer := &http.Server{
		Addr:         addr,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return NewServer
}
