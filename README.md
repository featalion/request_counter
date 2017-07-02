# Request Counter

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
