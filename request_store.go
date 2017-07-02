package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

type RequestInfo struct {
	RequestedAt   int64  `json:"requested_at"`
	RemoteAddress string `json:"remote_address"`
}

func (ri RequestInfo) IsOlderThan(nanos int64) bool {
	return ri.RequestedAt < nanos
}

func NewRequestInfo(req *http.Request) RequestInfo {
	return RequestInfo{
		RequestedAt:   time.Now().UnixNano(),
		RemoteAddress: req.RemoteAddr,
	}
}

type RequestStore struct {
	window   int64 // in nanoseconds
	list     *list.List
	mux      sync.Mutex
	ch       chan RequestInfo
	filename string
}

func (rs *RequestStore) evictOld() {
	lowBound := time.Now().UnixNano() - rs.window
	for elem := rs.list.Back(); elem != nil; elem = rs.list.Back() {
		if !elem.Value.(RequestInfo).IsOlderThan(lowBound) {
			break
		}
		rs.list.Remove(elem)
	}
}

func (rs *RequestStore) processInput() {
	var ri RequestInfo
	for {
		ri = <-rs.ch
		rs.mux.Lock()
		rs.list.PushFront(ri)
		rs.mux.Unlock()
	}
}

func (rs *RequestStore) LogRequest(req *http.Request) {
	rs.ch <- NewRequestInfo(req)
}

func (rs *RequestStore) Len() int {
	rs.mux.Lock()
	rs.evictOld()
	rs.mux.Unlock()

	return rs.list.Len()
}

func (rs *RequestStore) Dump() error {
	if rs.filename == "" {
		return nil
	}

	rs.mux.Lock()
	defer rs.mux.Unlock()

	rs.evictOld()
	requestInfos := make([]RequestInfo, rs.list.Len())
	for i, elem := 0, rs.list.Back(); elem != nil; i, elem = i+1, elem.Prev() {
		requestInfos[i] = elem.Value.(RequestInfo)
	}

	data, err := json.Marshal(requestInfos)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot marshall JSON:", err)
		return err
	}

	if err = ioutil.WriteFile(rs.filename, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write file %s: %s\n", rs.filename, err.Error())
		return err
	}

	return nil
}

func (rs *RequestStore) Load() {
	if rs.filename == "" {
		return
	}

	rs.mux.Lock()
	defer rs.mux.Unlock()

	data, err := ioutil.ReadFile(rs.filename)
	if err != nil {
		fmt.Printf("Cannot read file: %s\n", err.Error())
		return
	}

	var requestInfos []RequestInfo
	err = json.Unmarshal(data, &requestInfos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot unmarshall JSON: %s\n", err.Error())
		return
	}

	rs.list.Init()
	lowBound := time.Now().UnixNano() - rs.window
	for i := 0; i < len(requestInfos); i++ {
		if !requestInfos[i].IsOlderThan(lowBound) {
			rs.list.PushFront(requestInfos[i])
		}
	}
}

func NewRequestStore(windowLength int64, storeFilename string) *RequestStore {
	rs := &RequestStore{
		window:   windowLength * int64(time.Second),
		list:     list.New(),
		ch:       make(chan RequestInfo),
		filename: storeFilename,
	}
	rs.Load()

	go rs.processInput()

	return rs
}
