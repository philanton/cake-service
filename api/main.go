package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
    "github.com/philanton/cake-service/pkg/jwt"
)

func getCakeHandler(w http.ResponseWriter, r *http.Request, u User) {
	w.Write([]byte(u.FavoriteCake))
}

func wrapJWT(
	jwtService *jwt.JWTService,
	f func(http.ResponseWriter, *http.Request, *jwt.JWTService),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		f(rw, r, jwtService)
	}
}

func main() {
	r := mux.NewRouter()

	userService := UserService{
		repository: NewInMemoryUserStorage(),
	}

	jwtService, err := jwt.NewJWTService()
	if err != nil {
		panic(err)
	}

	r.HandleFunc(
		"/user/me",
		logRequest(jwtService.jwtAuth(userService.repository, getCakeHandler)),
	).Methods(http.MethodGet)
	r.HandleFunc("/user/register", logRequest(userService.Register)).Methods(http.MethodPost)
	r.HandleFunc(
		"/user/jwt",
		logRequest(wrapJWT(jwtService, userService.JWT)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/user/favorite_cake",
		logRequest(jwtService.jwtAuth(userService.repository, userService.OverwriteCake)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/user/password",
		logRequest(jwtService.jwtAuth(userService.repository, userService.OverwritePassword)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/user/email",
		logRequest(jwtService.jwtAuth(userService.repository, userService.OverwriteEmail)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/admin/ban",
		logRequest(jwtService.jwtAuth(userService.repository, userService.BanUser)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/admin/unban",
		logRequest(jwtService.jwtAuth(userService.repository, userService.UnbanUser)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/admin/inspect",
		logRequest(jwtService.jwtAuth(userService.repository, userService.History)),
	).Methods(http.MethodGet)

	srv := http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Println("Server started, hit Ctrl+C to stop")
	err = srv.ListenAndServe()
	if err != nil {
		log.Println("Server exited with error:", err)
	}

	log.Println("Good bye :)")
}
