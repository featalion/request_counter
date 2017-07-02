package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func logRequests(rs *RequestStore, n int) {
	for i := 0; i < n; i++ {
		rs.LogRequest(&http.Request{RemoteAddr: "0.0.0.0:8888"})
	}
	// ensure all requests are processed
	// TODO: implement a wrapper around time package
	for l := len(rs.ch); l > 0; l = len(rs.ch) {
		time.Sleep(1 * time.Millisecond)
	}
	time.Sleep(1 * time.Millisecond)
}

func assertRequestsCount(t *testing.T, testName string, rs *RequestStore, expected int) {
	if l := rs.Len(); l != expected {
		t.Errorf("%s: Expected store length of %d, but has %d\n", testName, expected, l)
	}
}

func TestRequestStoreBasic(t *testing.T) {
	// because we play with real time, speed up tests with parallel execution
	t.Parallel()
	rs := NewRequestStore(1, "")
	assertRequestsCount(t, "InitStore", rs, 0)

	logRequests(rs, 5)
	assertRequestsCount(t, "AfterLog", rs, 5)

	time.Sleep(1001 * time.Millisecond)
	assertRequestsCount(t, "AfterTimeout", rs, 0)
}

func TestRequestStoreAsync(t *testing.T) {
	t.Parallel()
	rs := NewRequestStore(1, "")
	assertRequestsCount(t, "InitStore", rs, 0)

	goN := 4
	reqN := 5
	c := make(chan bool, goN)
	for i := 0; i < goN; i++ {
		go func() {
			logRequests(rs, reqN)
			c <- true
		}()
	}
	for i := 0; i < goN; i++ {
		<-c
	}
	assertRequestsCount(t, "AfterAsyncLog", rs, goN*reqN)

	time.Sleep(1001 * time.Millisecond)
	assertRequestsCount(t, "AfterTimeout", rs, 0)
}

func TestRequestStorePersistence(t *testing.T) {
	t.Parallel()
	filename := "/tmp/rs.json"
	_ = os.Remove(filename) // ignore if file doesn't exist

	rs := NewRequestStore(2, filename)
	assertRequestsCount(t, "InitStore", rs, 0)

	logRequests(rs, 5)
	assertRequestsCount(t, "AfterLog", rs, 5)

	rs.Dump()
	var err error
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("File %s does not exists after dumping to it\n", filename)
	}

	rs.Load()
	assertRequestsCount(t, "LoadFromDump", rs, 5)

	emptyJson := []byte("[]")
	if err = ioutil.WriteFile(filename, emptyJson, 0600); err != nil {
		t.Errorf("Failed to write test JSON into file %s\n", filename)
	}

	rs.Load()
	assertRequestsCount(t, "LoadEmpty", rs, 0)

	logRequests(rs, 5)
	assertRequestsCount(t, "LogAgain", rs, 5)

	rs.Dump()
	time.Sleep(2001 * time.Millisecond)
	rs.Load()
	assertRequestsCount(t, "LoadTimeouted", rs, 0)

	if err = os.Remove(filename); err != nil {
		t.Errorf("Cannot remove test file %s: %s\n", filename, err.Error())
	}
}
