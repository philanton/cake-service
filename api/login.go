package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
)

type JWTParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (u *UserService) JWT(w http.ResponseWriter, r *http.Request, jwtService *MyJWTService) {
	params := &JWTParams{}
	err := json.NewDecoder(r.Body).Decode(params)
	if err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	passwordDigest := md5.New().Sum([]byte(params.Password))
	user, err := u.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if string(passwordDigest) != user.PasswordDigest {
		handleError(errors.New("invalid login params"), w)
		return
	}

	token, err := jwtService.GenerateJWT(user.Email)
	if err != nil {
		handleError(errors.New("invalid login params"), w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(token))
}
