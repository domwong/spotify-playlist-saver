package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/zmb3/spotify"

	sps "github.com/domwong/spotify-playlist-saver"

	"golang.org/x/oauth2"
)

var (
	redirectURI string
	saveARN     string // ARN of the save lambda
	auth        spotify.Authenticator
	state       = "abc123"
	errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
)

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if e := req.QueryStringParameters["error"]; e != "" {
		return serverError(errors.New("spotify: auth failed - " + e))
	}
	code := req.QueryStringParameters["code"]
	if code == "" {
		return serverError(errors.New("spotify: didn't get access code"))
	}
	actualState := req.QueryStringParameters["state"]
	if actualState != state {
		return serverError(errors.New("spotify: redirect state parameter doesn't match"))
	}
	tok, err := auth.Exchange(code)
	if err != nil {
		return serverError(err)
	}
	client := auth.NewClient(tok)
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1")},
	)
	if err != nil {
		return serverError(err)
	}
	// store tok
	ue, err := storeToken(sess, user.ID, tok)
	if err != nil {
		return serverError(err)
	}

	// TODO create cloudwatch event to cron trigger the /save endpoint
	if err := createCloudWatchCron(sess, ue); err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Success!",
	}, nil
}

func createCloudWatchCron(sess *session.Session, ue *sps.UserEntry) error {
	svc := cloudwatchevents.New(sess)

	t := time.Now().UTC().Add(2 * time.Minute) // schedule for 2 minutes in future
	ruleName := fmt.Sprintf("SpotifyWeekly-%s", ue.Username)
	_, err := svc.PutRule(&cloudwatchevents.PutRuleInput{
		Name:               aws.String(ruleName),
		ScheduleExpression: aws.String(fmt.Sprintf("cron(%d %d ? * %d *)", t.Minute(), t.Hour(), (t.Weekday() + 1))),
	})
	if err != nil {
		return err
	}
	_, err = svc.PutTargets(&cloudwatchevents.PutTargetsInput{
		Rule: aws.String(ruleName),
		Targets: []*cloudwatchevents.Target{
			&cloudwatchevents.Target{
				Id:    aws.String("save"),
				Input: aws.String(fmt.Sprintf(`{"username":"%s"}`, ue.Username)),
				Arn:   aws.String(saveARN),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func storeToken(sess *session.Session, username string, tok *oauth2.Token) (*sps.UserEntry, error) {
	// Create DynamoDB client
	svc := dynamodb.New(sess)
	ue := sps.UserEntry{
		Username:  username,
		Token:     *tok,
		Playlists: []string{"Discover Weekly"}, // TODO make this configurable
	}
	av, err := dynamodbattribute.MarshalMap(ue)
	if err != nil {
		return nil, err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("spotify"),
	}

	if _, err = svc.PutItem(input); err != nil {
		return nil, err
	}
	return &ue, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {

	errorLogger.Println(err.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func main() {
	redirectURI = os.Getenv("REDIRECT_URI")
	if redirectURI == "" {
		panic("REDIRECT_URI not set")
	}
	saveARN = os.Getenv("SAVE_ARN")
	if saveARN == "" {
		panic("SAVE_ARN not set")
	}
	auth = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPrivate, spotify.ScopePlaylistReadPrivate)
	lambda.Start(handleRequest)
}
