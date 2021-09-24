package main

import (
	"net/http"
	"strings"
)

type ProtectedHandler func(rw http.ResponseWriter, r *http.Request, u User)

func (j *JWTService) jwtAuth(ur UserRepository, h ProtectedHandler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		auth, err := j.ParseJWT(token)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("unauthorized"))
			return
		}

		err = ur.IsBanned(auth.Email)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte(err.Error()))
			return
		}

		err = ur.CheckNotInDB(token)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte(err.Error()))
			return
		}

		user, err := ur.Get(auth.Email)
		if err != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("unauthorized"))
			return
		}

		h(rw, r, user)
	}
}
