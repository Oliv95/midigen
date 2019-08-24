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

	nbrReads := 0
	c := make(chan int)
	for _, file := range args {
		filePtr, err := os.Open(file)
		defer filePtr.Close()
		if err != nil {
			log.Println("error reading file: ", file)
		} else {
			nbrReads = nbrReads + 1
			reader := bufio.NewReader(filePtr)
			go populateGraph(&generator, reader, c)
		}
	}
	// wait for all go routines to finish
	for i := 0; i < nbrReads; i++ {
		<-c
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

func populateGraph(gen *midigen.MidiGen, reader io.Reader, c chan int) {
	midigen.PopulateGraph(gen, reader)
	c <- 1
}
