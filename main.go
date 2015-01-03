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
	installScriptURL = "https://raw.githubusercontent.com/Scalingo/cli/master/dists/install.sh"
	installScript    = &bytes.Buffer{}
	initOnce         = &sync.Once{}
	scriptLock       = &sync.Mutex{}
)

func main() {
	scriptReady := make(chan struct{})
	go scriptUpdater(scriptReady)
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		<-scriptReady
		res.WriteHeader(200)
		res.Header().Set("Content-Length", fmt.Sprintf("%d", installScript.Len()))
		res.Header().Set("Content-Type", "text/plain")
		scriptLock.Lock()
		io.Copy(res, installScript)
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
		scriptUpdate()
		initOnce.Do(func() {
			close(done)
		})
		<-timer.C
	}
}

func scriptUpdate() {
	req, err := http.Get(installScriptURL)
	if err != nil {
		log.Println("http get error:", err)
		return
	}
	defer req.Body.Close()
	installScript.Reset()
	scriptLock.Lock()
	n, err := io.Copy(installScript, req.Body)
	scriptLock.Unlock()
	if err != nil {
		log.Println("Copy error:", err)
		return
	}
	log.Println("New install script length", n)
}
