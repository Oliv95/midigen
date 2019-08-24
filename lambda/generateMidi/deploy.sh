#!/bin/sh

GOOS=linux GOARCH=amd64 go build -o main generateMidi.go
zip generateMidi.zip main

aws lambda update-function-code --function-name generateMidi --zip-file fileb://generateMidi.zip --profile midiDev --region eu-north-1
rm generateMidi.zip
