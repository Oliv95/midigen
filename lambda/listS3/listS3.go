package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"os"
	"strconv"
)

type fileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type response struct {
	Data []fileInfo `json:"data"`
}

type errorMsg struct {
	ErrorMsg string `json:"error"`
}

func handleRequest(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})

	responseHeaders := make(map[string]string)
	responseHeaders["Content-Type"] = "application/json"

	if err != nil {
		errorMsg := fmt.Sprintf("Error creating sessions %v", err)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	// Create S3 service client
	svc := s3.New(sess)

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(os.Getenv("BUCKET_NAME"))})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to list items in bucket %q, %v", os.Getenv("BUCKET_NAME"), err)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	limit, offset := parseParameters(&event)
	totalNbrOfFiles := len(resp.Contents)

	if limit < 0 || offset < 0 {
		errorMsg := fmt.Sprintf("limit or offset negative: limit %v offset %v\n", limit, offset)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	if offset > totalNbrOfFiles {
		errorMsg := fmt.Sprintf("Offset cannot be greater than total nbr of files. offset: %v total nbr of files %v", offset, totalNbrOfFiles)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil

	}

	fileInfos := []fileInfo{}
	start := offset
	stop := minInt(offset+limit, totalNbrOfFiles)

	// get the requested range
	for _, item := range resp.Contents[start:stop] {
		fileInfos = append(fileInfos, fileInfo{*item.Key, *item.Size})
	}

	serverResponse := response{Data: fileInfos}
	responseBody, err := json.Marshal(&serverResponse)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to serilize response %v %v", serverResponse, err)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil

	}

	return createResponse(200, responseHeaders, string(responseBody)), nil
}

func createResponse(statusCode int, headers map[string]string, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: headers, Body: body, IsBase64Encoded: true}
}

func stringToJSON(msg string) string {
	jsonString, err := json.Marshal(&errorMsg{ErrorMsg: msg})
	if err != nil {
		errorMsg := fmt.Sprintf("{ \"error\": Failed to serilize response %v %v", msg)
		fmt.Println(errorMsg)
		return errorMsg
	}
	return string(jsonString)

}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseParameters(event *events.APIGatewayProxyRequest) (int, int) {
	limit, err := strconv.Atoi(event.QueryStringParameters["limit"])
	if err != nil {
		limit = 0
	}
	offset, err := strconv.Atoi(event.QueryStringParameters["offset"])
	if err != nil {
		offset = 0
	}
	return limit, offset

}

func main() {
	lambda.Start(handleRequest)

}
