# heartbeat

Simple HTTP heartbeat implementation for Golang. This implementation works particularly well with [Uptime Kuma](https://github.com/louislam/uptime-kuma)'s push monitors.

## Installation

```shell
go get github.com/cdzombak/heartbeat
```

## Usage

Create and start a Heartbeat client:

```go
hb, err = NewHeartbeat(&HeartbeatConfig{
    HeartbeatInterval: 30 * time.Second,
    LivenessThreshold: 60 * time.Second,
    HeartbeatURL:      "https://uptimekuma.example.com:9001/api/push/1234abcd?status=up&msg=OK&ping=",
    OnError: func(err error) {
        log.Printf("heartbeat error: %s", err)
    },
})
if err != nil {
    panic(err)
}

// other program setup might go here

hb.Start()
```

Then, in your program's main loop/ticker/event handler, call `Alive` periodically to indicate that everything's working:

```go
hb.Alive(time.Now())
```

## License

MIT; see `LICENSE` in this repository.

## Author

- Chris Dzombak ([dzombak.com](https://www.dzombak.com))
    - [GitHub @cdzombak](https://www.github.com/cdzombak)
