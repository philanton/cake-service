package main

import (
    "flag"
    "log"
    "net/http"
    "time"
)

var addr = flag.String("addr", ":8081", "http service address")

func timeGen(hub *Hub) {
    clock := time.NewTicker(5 * time.Second)

    for {
        timeNow := <- clock.C
        hub.broadcast <- []byte(timeNow.Format(time.UnixDate))
    }
}

func main() {
    flag.Parse()
    hub := NewHub()
    go hub.run()
    go timeGen(hub)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        serveWS(hub, w, r)
    })
    err := http.ListenAndServe(*addr, nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

