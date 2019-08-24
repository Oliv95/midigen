#!/bin/sh

GOOS=linux GOARCH=amd64 go build -o main listS3.go
zip main.zip main

aws lambda update-function-code --function-name listS3 --zip-file fileb://main.zip --profile midiDev --region eu-north-1
rm main.zip
