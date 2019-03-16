package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/zmb3/spotify"
)

var (
	redirectURI string
	auth        spotify.Authenticator
	state       = "abc123"
	errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
)

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// TODO generate and check state properly
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusTemporaryRedirect,
		Headers: map[string]string{
			"location": auth.AuthURL(state),
		},
	}, nil
}

func main() {
	redirectURI = os.Getenv("REDIRECT_URI")
	if redirectURI == "" {
		panic("REDIRECT_URI not set")
	}
	auth = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPrivate, spotify.ScopePlaylistReadPrivate)
	lambda.Start(handleRequest)
}
