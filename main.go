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

	http.HandleFunc("/robots.txt", func(res http.ResponseWriter, req *http.Request) {
		fileBytes, err := os.ReadFile("robots.txt")
		if err != nil {
			res.WriteHeader(500)
			return
		}
		res.WriteHeader(200)
		res.Header().Add("Content-Type", "text/plain")
		_, err = res.Write(fileBytes)
		if err != nil {
			res.WriteHeader(500)
			return
		}
	})
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
			writeShortResponse(res, http.StatusNotFound)
			return
		}

		reqVersion = sequences[1]
		if reqArchive == "" || reqVersion == "" {
			writeShortResponse(res, http.StatusNotFound)
			return
		}
		archiveURL := fmt.Sprintf(ghReleaseURL, reqVersion, reqArchive)

		githubRes, err := http.Get(archiveURL)
		if err != nil {
			log.Println("http get error:", err)
			writeShortResponse(res, http.StatusBadRequest)
			return
		}
		defer githubRes.Body.Close()

		if githubRes.StatusCode != http.StatusOK {
			writeShortResponse(res, githubRes.StatusCode)
			return
		}

		res.Header().Set("Content-Type", githubRes.Header.Get("Content-Type"))
		res.Header().Set("Content-Length", githubRes.Header.Get("Content-Length"))

		_, err = io.Copy(res, githubRes.Body)
		if err != nil {
			log.Println("io.Copy error:", err)
			writeShortResponse(res, http.StatusInternalServerError)
		}
	})

	port := "20205"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	fmt.Printf("Listen on 0.0.0.0:%s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func writeShortResponse(res http.ResponseWriter, code int) {
	res.WriteHeader(code)
	res.Write([]byte(http.StatusText(code)))
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
