# MCPing
A Golang library that facilitates Minecraft 1.7 server list pings.

[Godoc] (https://godoc.org/github.com/ammario/mcping)

## Install

`` go get github.com/ammario/mcping``

## Usage
```go
package main

import (
    "github.com/ammario/mcping"
    "fmt"
)

func main() {
    resp, err := mcping.Ping("us.mineplex.com:25565")
    fmt.Println("Mineplex has", resp.Online, "players online")
}
```

## Response Struct

The struct returned by the ``Ping()`` function has the following schema

```go
type PingResponse struct {
    Latency  uint   //Latency in ms
    Online   int    //Amount of online players
    Max      int    //Maximum amount of players
    Protocol int    //E.g '4'
    Favicon  string //Base64 encoded favicon in data URI format
    Motd     string
    Server   string //E.g 'PaperSpigot'
    Version  string //E.g "1.7.10"
    Sample   []PlayerSample
}
```

### PlayerSample struct

```go
type PlayerSample struct {
    UUID string //e.g "d8a973a5-4c0f-4af6-b1ea-0a76cd210cc5"
    Name string //e.g "Ammar"
}
```

## Future Plans
- Pre 1.7 ping
