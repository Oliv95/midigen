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

	// Collect all the files from the cmd-line arguments
	readers := []io.Reader{}
	for _, file := range args {
		filePtr, err := os.Open(file)
		defer filePtr.Close()
		if err != nil {
			log.Println("error reading file: ", file)
		} else {
			reader := bufio.NewReader(filePtr)
			readers = append(readers, reader)
		}
	}
	filePtr, err := os.Create("markov.mid")
	defer filePtr.Close()
	writer := bufio.NewWriter(filePtr)
	err = midigen.GenerateMidi(io.MultiReader(readers...), writer, 50)
	if err != nil {
		log.Fatalln(err)
	}
	writer.Flush()
}
