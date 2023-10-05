package main

import (
	"flag"
	"log"

	"github.com/yifeng-qiu/ytdlp_backend/internal/server"
	"github.com/yifeng-qiu/ytdlp_backend/pkg/downloader"
)

var addr = flag.String("addr", ":1718", "http service address")

func main() {
	flag.Parse()
	myServer := &server.RequestHandler{
		Requests:        make(map[string]server.Request),
		DownloadManager: downloader.NewDownloadManager(),
	}

	myhttpServer := myServer.NewHTTPServer(*addr)

	err := myhttpServer.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
