# Request Counter

Request counter is HTTP server, which responds to `/count` requests with number of request it receives in last N seconds (default is 60).
It stores data about requests in JSON file (default name is `rs.json`), tries to load file on start, and writes data to the file before exit, including handling interrupting signals from OS.

## Build

To build the project simply use `go build`.

## Use

To get help

```
$ ./request_counter -h
```

The HTTP server responds requests only to path `/count`. For any other URI it returns HTTP 404.

```
$ ./request_counter
$ curl http://localhost:8080/count
0
$ curl http://localhost:8080/count
1
```

## Test

`go test`
