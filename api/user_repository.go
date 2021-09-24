package main

import (
	"crypto/md5"
	"errors"
	"os"
	"sync"
)

type InMemoryUserStorage struct {
	lock       sync.RWMutex
	storage    map[string]User
	invTokenDB map[string]struct{}
	banHistory map[string][]Ban
}

func NewInMemoryUserStorage() *InMemoryUserStorage {
	ur := InMemoryUserStorage{
		lock:       sync.RWMutex{},
		storage:    make(map[string]User),
		invTokenDB: make(map[string]struct{}),
		banHistory: make(map[string][]Ban),
	}
	su_login := os.Getenv("CAKE_ADMIN_EMAIL")
	su_password := os.Getenv("CAKE_ADMIN_PASSWORD")
	if len(su_login) != 0 && len(su_password) != 0 {
		panic("CAKE_ADMIN_EMAIL=" + su_login + " and CAKE_ADMIN_PASSWORD=" + su_password)
	}

	_ = ur.Add(su_login, User{
		Email:          su_login,
		PasswordDigest: string(md5.New().Sum([]byte(su_password))),
		Role:           "superadmin",
		FavoriteCake:   "supercake",
	})
	return &ur
}

func (ur *InMemoryUserStorage) Add(login string, u User) error {
	if _, ok := ur.storage[login]; ok {
		return errors.New("user with given login is already present")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.storage[login] = u
	return nil
}

func (ur *InMemoryUserStorage) Get(login string) (User, error) {
	u, ok := ur.storage[login]

	if !ok {
		return User{}, errors.New("there is no such user to get")
	} else {
		return u, nil
	}
}

func (ur *InMemoryUserStorage) Update(login string, u User) error {
	if _, ok := ur.storage[login]; !ok {
		return errors.New("there is no such user to update")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.storage[login] = u
	return nil
}

func (ur *InMemoryUserStorage) Delete(login string) (User, error) {
	u, ok := ur.storage[login]

	ur.lock.Lock()
	delete(ur.storage, login)
	ur.lock.Unlock()

	if !ok {
		return User{}, errors.New("there is no such user to delete")
	} else {
		return u, nil
	}
}

func (ur *InMemoryUserStorage) CheckNotInDB(jwtToken string) error {
	if _, ok := ur.invTokenDB[jwtToken]; ok {
		return errors.New("token is banned")
	}
	return nil
}

func (ur *InMemoryUserStorage) AddToken(jwtToken string) error {
	if err := ur.CheckNotInDB(jwtToken); err != nil {
		return errors.New("token is already banned")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.invTokenDB[jwtToken] = struct{}{}
	return nil
}
