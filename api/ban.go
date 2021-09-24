package main

import (
	"errors"
	"time"
)

type Ban struct {
	BannedAt    time.Time
	WhoBanned   string
	UnbannedAt  time.Time
	WhoUnbanned string
	Reason      string
}

func (ur *InMemoryUserStorage) IsBanned(login string) error {
	history, ok := ur.banHistory[login]
	if !ok {
		return nil
	}

	lastBan := history[len(history)-1]
	if lastBan.UnbannedAt.IsZero() {
		return errors.New("user is banned with reason \"" + lastBan.Reason + "\" by \"" + lastBan.WhoBanned + "\"")
	}

	return nil
}

func (ur *InMemoryUserStorage) BanHistory(login string) ([]Ban, error) {
	history, ok := ur.banHistory[login]
	if !ok {
		return []Ban{}, errors.New("user history is clear")
	}

	return history, nil
}

func (ur *InMemoryUserStorage) Ban(login string, byLogin string, reason string) error {
	history, ok := ur.banHistory[login]
	if ok {
		lastBan := history[len(history)-1]
		if lastBan.UnbannedAt.IsZero() {
			return errors.New("user is already banned")
		}
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.banHistory[login] = append(history, Ban{
		BannedAt:    time.Now(),
		WhoBanned:   byLogin,
		UnbannedAt:  time.Time{},
		Reason:      reason,
		WhoUnbanned: "",
	})

	return nil
}

func (ur *InMemoryUserStorage) Unban(login string, byLogin string) error {
	history, ok := ur.banHistory[login]
	if !ok {
		return errors.New("user history is clear")
	}

	lastBan := history[len(history)-1]
	if !lastBan.UnbannedAt.IsZero() {
		return errors.New("user is not banned")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	lastBan.UnbannedAt = time.Now()
	lastBan.WhoUnbanned = byLogin
	history[len(history)-1] = lastBan

	return nil
}
