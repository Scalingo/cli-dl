package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	versionURL       = "https://raw.githubusercontent.com/Scalingo/cli/master/VERSION"
	version          = &bytes.Buffer{}
	installScriptURL = "https://raw.githubusercontent.com/Scalingo/cli/master/dists/install.sh"
	ghReleaseURL     = "https://github.com/Scalingo/cli/releases/download/%s/%s"
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
	http.HandleFunc("/release/", func(res http.ResponseWriter, req *http.Request) {
		<-scriptReady
		reqArchive := ""
		reqVersion := ""

		fmt.Sscanf(req.URL.Path, "/release/%s", &reqArchive)
		reqArchive = strings.Replace(reqArchive, "latest", strings.TrimRight(version.String(), "\n"), 1)

		sequences := strings.Split(reqArchive, "_")
		if len(sequences) != 4 {
			writeShortResponse(res, http.StatusNotFound, "Not found")
			return
		}

		reqVersion = sequences[1]
		if reqArchive == "" || reqVersion == "" {
			writeShortResponse(res, http.StatusNotFound, "Not found")
			return
		}
		archiveUrl := fmt.Sprintf(ghReleaseURL, reqVersion, reqArchive)

		githubRes, err := http.Get(archiveUrl)
		if err != nil {
			log.Println("http get error:", err)
			writeShortResponse(res, http.StatusBadRequest, "Bad request")
			return
		}
		defer githubRes.Body.Close()

		if githubRes.StatusCode != http.StatusOK {
			writeShortResponse(res, http.StatusBadRequest, "Bad request")
			return
		}

		res.Header().Set("Content-Type", req.Header.Get("Content-Type"))
		res.Header().Set("Content-Length", req.Header.Get("Content-Length"))
		_, err = io.Copy(res, githubRes.Body)
		if err != nil {
			log.Println("io.Copy error:", err)
			writeShortResponse(res, http.StatusInternalServerError, "Internal error")
		}
	})

	port := "4000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	fmt.Printf("Listen on 0.0.0.0:%s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func writeShortResponse(res http.ResponseWriter, code int, content string) {
	res.WriteHeader(code)
	if content != "" {
		res.Write([]byte(content))
	}
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
