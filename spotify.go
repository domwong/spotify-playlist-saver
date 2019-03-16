package sps

import (
	"golang.org/x/oauth2"
)

type UserEntry struct {
	Username  string       `json:"username"`
	Token     oauth2.Token `json:"token"`
	Playlists []string     `json:"playlists"`
}
