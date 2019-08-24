package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Oliv95/midigen"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
)

type Event struct {
	Keys              []string `json:"keys"`
	ResultingFileName string   `json:"resultingFileName"`
	Iterations        int      `json:"iterations"`
}

type Response struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

type ErrorMsg struct {
	ErrorMsg string `json:"error"`
}

func HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	responseHeaders := make(map[string]string)
	responseHeaders["Content-Type"] = "application/json"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})

	if err != nil {
		errorMsg := fmt.Sprintf("Error creating sessions %v", err)
		fmt.Println(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	// read data from event, should be in a different function
	data := &Event{}
	err = json.Unmarshal([]byte(event.Body), data)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to parse body %v", event.Body)
		fmt.Printf(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	keys := data.Keys

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)

	readersChan := make(chan io.Reader)
	for _, key := range keys {
		fmt.Println(key)
		go downloadMidi(key, downloader, readersChan)
	}

	generator := midigen.EmptyGenerator()
	fmt.Printf("keys = %v len(keys) = %v\n", keys, len(keys))
	for i := 0; i < len(keys); i++ {
		reader := <-readersChan
		fmt.Printf("recived from chan %v", reader)
		if reader != nil {
			err = midigen.PopulateGraph(&generator, reader)
			if err != nil {
				errorMsg := fmt.Sprintf("Error populate graph: %v", err)
				fmt.Printf(errorMsg)
				return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
			}
		}
	}

	uploader := s3manager.NewUploader(sess)

	resultingFileName := data.ResultingFileName
	iterations := data.Iterations
	midiRead, midiWrite := io.Pipe()

	go func() {
		err := midigen.GenerateMidi(&generator, midiWrite, iterations)
		if err != nil {
			errorMsg := "Unable to generate midi\n"
			fmt.Printf(errorMsg)
		}
		midiWrite.Close()
	}()

	err = uploadMidi(resultingFileName, midiRead, uploader)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to upload file %v", resultingFileName)
		fmt.Printf(errorMsg)
		return createResponse(500, responseHeaders, stringToJSON(errorMsg)), nil
	}

	return createResponse(200, responseHeaders, "{\"testing\": \"bla\"}"), nil

}

func uploadMidi(s3key string, data io.Reader, uploader *s3manager.Uploader) error {
	fmt.Printf("starting upload of file %v\n", s3key)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("BUCKET_NAME_GEN")),
		Key:    aws.String(s3key),
		Body:   data,
	})
	if err != nil {
		return err
	}
	return nil
}

func downloadMidi(s3key string, downloader *s3manager.Downloader, c chan io.Reader) {
	// Create a buffer to write the S3 Object contents to.
	buf := aws.NewWriteAtBuffer([]byte{})

	// Write the contents of S3 Object to the file
	n, err := downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key:    aws.String(s3key),
	})
	fmt.Printf("n = %v buf = %v err = %v\n", n, buf, err)
	if err != nil {
		fmt.Println("Error during download " + err.Error())
		c <- nil
		return
	}

	reader := bytes.NewReader(buf.Bytes())
	c <- reader
}

func createResponse(statusCode int, headers map[string]string, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: headers, Body: body, IsBase64Encoded: true}
}

func stringToJSON(msg string) string {
	jsonString, err := json.Marshal(&ErrorMsg{ErrorMsg: msg})
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

func main() {
	lambda.Start(HandleRequest)

}
