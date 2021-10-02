package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

func (us *UserService) getCakeHandler(w http.ResponseWriter, r *http.Request, u User) {
	w.Write([]byte(u.FavoriteCake))
	cakesGiven.Inc()
}

func wrapJWT(
	jwtService *MyJWTService,
	f func(http.ResponseWriter, *http.Request, *MyJWTService),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		f(rw, r, jwtService)
	}
}

func main() {
	r := mux.NewRouter()

	userService := UserService{
		notifier:   make(chan []byte, 10),
		repository: NewInMemoryUserStorage(),
	}

	myJWTService, err := NewMyJWTService()
	if err != nil {
		panic(err)
	}

	go runPublisher(userService.notifier)
	go startProm()

	r.HandleFunc(
		"/user/me",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.getCakeHandler)),
	).Methods(http.MethodGet)
	r.HandleFunc("/user/register", logRequest(userService.Register)).Methods(http.MethodPost)
	r.HandleFunc(
		"/user/jwt",
		logRequest(wrapJWT(myJWTService, userService.JWT)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/user/favorite_cake",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.OverwriteCake)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/user/password",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.OverwritePassword)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/user/email",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.OverwriteEmail)),
	).Methods(http.MethodPut)
	r.HandleFunc(
		"/admin/ban",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.BanUser)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/admin/unban",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.UnbanUser)),
	).Methods(http.MethodPost)
	r.HandleFunc(
		"/admin/inspect",
		logRequest(myJWTService.jwtAuth(userService.repository, userService.History)),
	).Methods(http.MethodGet)

    apiPort := os.Getenv("API_PORT")
	srv := http.Server{
		Addr:    ":"+apiPort,
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
