package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type CakeOverwriteParams struct {
	FavoriteCake string `json:"favorite_cake"`
}

type PasswordOverwriteParams struct {
	Password string `json:"password"`
}

type EmailOverwriteParams struct {
	Email string `json:"email"`
}

func (us *UserService) OverwriteCake(w http.ResponseWriter, r *http.Request, u User) {
	params := &CakeOverwriteParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	if err := validateCake(params.FavoriteCake); err != nil {
		handleError(err, w)
		return
	}

	u.FavoriteCake = params.FavoriteCake
	if err := us.repository.Update(u.Email, u); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("favorite cake changed"))
    us.notifier <- []byte("updated cake: " + u.Email)
}

func (us *UserService) OverwritePassword(w http.ResponseWriter, r *http.Request, u User) {
	params := &PasswordOverwriteParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	if err := validatePassword(params.Password); err != nil {
		handleError(err, w)
		return
	}

	u.PasswordDigest = string(md5.New().Sum([]byte(params.Password)))
	if err := us.repository.Update(u.Email, u); err != nil {
		handleError(err, w)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := us.repository.AddToken(token); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("password changed"))
    us.notifier <- []byte("updated password: " + u.Email)
}

func (us *UserService) OverwriteEmail(w http.ResponseWriter, r *http.Request, u User) {
	params := &EmailOverwriteParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	if err := validateEmail(params.Email); err != nil {
		handleError(err, w)
		return
	}

	if newU, err := us.repository.Delete(u.Email); err != nil {
		handleError(err, w)
		return
	} else {
		newU.Email = params.Email
		if err = us.repository.Add(newU.Email, newU); err != nil {
			handleError(err, w)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("email changed"))
    us.notifier <- []byte("updated email: " + u.Email + " -> " + params.Email)
}
