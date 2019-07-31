package main

import (
	"bufio"
	"github.com/Oliv95/midigen"
	"io"
	"log"
	"os"
)

func main() {
	args := os.Args[1:]
	generator := midigen.EmptyGenerator()

	for _, file := range args {
		filePtr, err := os.Open(file)
		defer filePtr.Close()
		if err != nil {
			log.Println("error reading file: ", file)
		} else {
			reader := bufio.NewReader(filePtr)
			midigen.PopulateGraph(&generator, reader)
		}
	}
	filePtr, err := os.Create("markov.mid")
	defer filePtr.Close()
	writer := bufio.NewWriter(filePtr)
	err = midigen.GenerateMidi(&generator, writer, 1000)
	if err != nil {
		log.Fatalln(err)
	}
	writer.Flush()
}
