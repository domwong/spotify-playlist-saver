# Spotify Playlist Saver

The Discover Weekly playlist on Spotify is pretty awesome, but it gets updated every week so you unless you save all the songs you lose those hidden gems. This is a toy project to automatically and regularly save Spotify playlists. There is an existing IFTTT integration but I wanted to flex my coding chops and try out some things on AWS.

## Architecture
Consists of:
- Three AWS lambdas 
  - `root` - a simple redirect which kicks off the auth flow.
  - `callback` - which does the OAuth dance, retrieving the OAuth token and storing in DynamoDB for subsequent operations.
  - `save` - which does the actual saving of the playlist. Configure a cloudwatch cron job to run it once a week.
- A DynamoDB table
- Some cloud watch rules to schedule weekly saving

## Usage
Hit the root `/` endpoint, which will redirect you to Spotify to login and authorise the app. Once you've authorised the app you should see a success message. This will schedule a weekly back up of Discover Weekly, the first one will happen in 2 minutes from when you invoke the app.

## Build and Deploy
1. Create DynamoDB table `spotify` with `username` as primary key
1. Run `build.sh` to build and deploy the lambda code to AWS.
1. Create API gateway definition 
   1. Point `/` to root
   1. Point `/callback` to callback and add query string parameters `code`, `state`, and `error`
1. Deploy the API
1. Update lambdas with env variables
   1. `REDIRECT_URI` - this is the API Gateway url to the `/callback` endpoint
   1. `SPOTIFY_ID` - the client ID from Spotify 
   1. `SPOTIFY_SECRET` - the client secret from Spotify
1. Create an IAM role for the lambdas to run under. They need to
   1. Create new cloud watch rules
   1. Read and write to dynamo db
   1. Write logs to cloud watch


## TODO
- Add support for the state parameter to fix security hole