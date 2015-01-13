package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	versionURL       = "https://raw.githubusercontent.com/Scalingo/appsdeck-executables/master/latest"
	version          = &bytes.Buffer{}
	installScriptURL = "https://raw.githubusercontent.com/Scalingo/cli/master/dists/install.sh"
	installScript    = &bytes.Buffer{}
	initOnce         = &sync.Once{}
	scriptLock       = &sync.Mutex{}
)

func main() {
	scriptReady := make(chan struct{})
	go scriptUpdater(scriptReady)
	http.HandleFunc("/version", func(res http.ResponseWriter, req *http.Request) {
		<-scriptReady
		res.Header().Set("Content-Length", fmt.Sprintf("%d", version.Len()))
		res.Header().Set("Content-Type", "text/plain")
		scriptLock.Lock()
		res.Write(version.Bytes())
		scriptLock.Unlock()
	})
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		<-scriptReady
		res.WriteHeader(200)
		res.Header().Set("Content-Length", fmt.Sprintf("%d", installScript.Len()))
		res.Header().Set("Content-Type", "text/plain")
		scriptLock.Lock()
		res.Write(installScript.Bytes())
		scriptLock.Unlock()
	})

	port := "4000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	fmt.Printf("Listen on 0.0.0.0:%s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func scriptUpdater(done chan struct{}) {
	timer := time.NewTimer(time.Hour)
	for {
		go update(installScript, installScriptURL)
		go update(version, versionURL)
		initOnce.Do(func() {
			close(done)
		})
		<-timer.C
	}
}

func update(buf *bytes.Buffer, u string) {
	req, err := http.Get(u)
	if err != nil {
		log.Println("http get error:", err)
		return
	}
	defer req.Body.Close()
	buf.Reset()
	scriptLock.Lock()
	_, err = io.Copy(buf, req.Body)
	scriptLock.Unlock()
	if err != nil {
		log.Println("Copy error:", err)
		return
	}
}
