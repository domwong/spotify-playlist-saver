package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/zmb3/spotify"

	sps "github.com/domwong/spotify-playlist-saver"
)

var (
	redirectURI string
	auth        spotify.Authenticator
	errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
)

type CronEvent struct {
	Username string `json:"username"`
}

func handleRequest(ctx context.Context, ce CronEvent) (string, error) {
	// read token for user

	username := ce.Username
	if username == "" {
		return serverError(errors.New("spotify: didn't get username"))
	}

	// retrieve tok
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1")},
	)

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	res, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("spotify"),
		Key: map[string]*dynamodb.AttributeValue{
			"username": {
				S: aws.String(username),
			},
		},
	})

	ue := sps.UserEntry{}
	if err = dynamodbattribute.UnmarshalMap(res.Item, &ue); err != nil {
		return serverError(err)
	}
	client := auth.NewClient(&ue.Token)
	if err := savePlaylists(&client, &ue); err != nil {
		return serverError(err)
	}

	return "", nil
}

func savePlaylists(client *spotify.Client, userEntry *sps.UserEntry) error {
	limit := 50
	off := 0
	for {
		pls, err := client.CurrentUsersPlaylistsOpt(&spotify.Options{
			Limit:  &limit,
			Offset: &off,
		})

		if err != nil {
			return err
		}
		for _, v := range pls.Playlists {
			found := false
			for _, p := range userEntry.Playlists {
				if v.Name == p {
					found = true
					break
				}
			}
			if !found {
				continue
			}

			pl, err := client.GetPlaylist(v.ID)
			if err != nil {
				return err
			}
			plTracks := make([]spotify.ID, len(pl.Tracks.Tracks))
			for i, tr := range pl.Tracks.Tracks {
				plTracks[i] = tr.Track.ID
			}

			cpl, err := client.CreatePlaylistForUser(userEntry.Username, fmt.Sprintf("%s %s", v.Name, time.Now().Format("2006-01-02")), "Autosaved snapshot", false)
			if err != nil {
				return err
			}

			if _, err = client.AddTracksToPlaylist(cpl.ID, plTracks...); err != nil {
				return err
			}

		}
		if len(pls.Playlists) < limit {
			break
		}
		off++

	}
	return nil
}

func serverError(err error) (string, error) {
	errorLogger.Println(err.Error())

	return "", err
}

func main() {
	redirectURI = os.Getenv("REDIRECT_URI")
	if redirectURI == "" {
		panic("REDIRECT_URI not set")
	}
	auth = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPrivate, spotify.ScopePlaylistReadPrivate)
	lambda.Start(handleRequest)
}
