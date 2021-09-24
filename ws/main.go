package main

import (
    "flag"
    "log"
    "net/http"
    "time"

    "github.com/philanton/cake-service/pkg/jwt"
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

    jwtService := jwt.NewJWTService()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if _, err := jwtService.ParseJWT(token); err != nil {
            w.WriteHeader(401)
            w.Write("unauthorized")
            return
        }

        serveWS(hub, w, r)
    })
    err := http.ListenAndServe(*addr, nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

