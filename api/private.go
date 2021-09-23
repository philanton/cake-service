package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type BanUserParams struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

type UnbanUserParams struct {
	Email string `json:"email"`
}

func (us *UserService) BanUser(w http.ResponseWriter, r *http.Request, u User) {
	params := &BanUserParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("not enough privileges"), w)
		return
	}

	err = us.repository.Ban(params.Email, u.Email, params.Reason)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("user \"" + params.Email + "\" is banned with reason\"" + params.Reason + "\" by \"" + u.Email + "\""))
	us.notifier <- []byte("banned: " + params.Email)
}

func (us *UserService) UnbanUser(w http.ResponseWriter, r *http.Request, u User) {
	params := &UnbanUserParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("could not read params"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("not enough privileges"), w)
		return
	}

	err = us.repository.Unban(params.Email, u.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("user \"" + params.Email + "\" is unbanned by \"" + u.Email + "\""))
	us.notifier <- []byte("unbanned: " + params.Email)
}

func (us *UserService) History(w http.ResponseWriter, r *http.Request, u User) {
	email := r.URL.Query().Get("email")
	if len(email) == 0 {
		handleError(errors.New("email is not specified"), w)
		return
	}

	user, err := us.repository.Get(email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("not enough privileges"), w)
		return
	}

	history, err := us.repository.BanHistory(email)
	if err != nil {
		handleError(err, w)
		return
	}

	body, err := json.Marshal(history)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
