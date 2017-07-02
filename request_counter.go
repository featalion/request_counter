package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

const (
	DefaultRequestStoreWindowSec = 60
	DefaultRequestStoreFileName  = "rs.json"
	DefaultAddress               = ":8080"
)

type CountHandler struct {
	rs *RequestStore
}

func (h *CountHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.String() != "/count" {
		resp.WriteHeader(http.StatusNotFound)
		io.WriteString(resp, "Not Found\n")
		return
	}

	count := h.rs.Len()
	h.rs.LogRequest(req)

	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	io.WriteString(resp, fmt.Sprintf("%d\n", count))
}

func main() {
	var (
		window    int64  = DefaultRequestStoreWindowSec
		filestore string = DefaultRequestStoreFileName
		address   string = DefaultAddress
		err       error
	)
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-f", "--file":
			if i+1 < len(os.Args) {
				filestore = os.Args[i+1]
				i++
			}
		case "-w", "--window":
			if i+1 < len(os.Args) {
				window, err = strconv.ParseInt(os.Args[i+1], 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cannot read window from '%s', set to default value %d sec\n", os.Args[i+1], DefaultRequestStoreWindowSec)
					window = DefaultRequestStoreWindowSec
				} else {
					i++
				}
			}
		case "-a", "--address":
			if i+1 < len(os.Args) {
				address = os.Args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Printf(`%s command line arguments:
-f, --file - name of JSON file, that represents file store for request counter, default is 'rs.json'
-w, --window - duration of moving window in seconds, default is 60
-a, --address - network address to bind server to, default is ':8080'
-h, --help - prints this help message and exists
`, os.Args[0])
			os.Exit(0)
		}
	}

	rs := NewRequestStore(window, filestore)
	defer rs.Dump()

	s := &http.Server{
		Addr:           address,
		Handler:        &CountHandler{rs: rs},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		s.Close()
	}()

	s.ListenAndServe()
}
