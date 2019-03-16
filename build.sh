#!/bin/bash

build_lambda() {
    lambda_name=$1
    GOOS=linux go build -o $lambda_name lambdas/$lambda_name/main.go
    rm $lambda_name.zip
    zip $lambda_name.zip $lambda_name
    aws lambda get-function --function-name $lambda_name
    if [[ $? -eq 0 ]]; then
        aws lambda update-function-code --function-name $lambda_name --zip-file fileb://$lambda_name.zip 
    else
        aws lambda create-function --region eu-west-1 --function-name $lambda_name --memory 128 --role arn:aws:iam::030283614624:role/LambdaDynamoDBSpotify --runtime go1.x --zip-file fileb://$lambda_name.zip --handler $lambda_name
    fi
}

build_lambda root
build_lambda callback
build_lambda save
