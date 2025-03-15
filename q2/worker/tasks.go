package main

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	// "slices"
	"strconv"
	"strings"

	mapReducepb "q2/protofiles"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func contains(slice []string, item string) bool {
    for _, val := range slice {
        if val == item {
            return true
        }
    }
    return false
}

func mergeAndReduceInvertedIndex(numMappers int, reducerId int) {
	wordToFile := make(map[string]([]string)) 
	for m := range numMappers {
		inputFile := fmt.Sprintf("mapper_output/map-%d-%d.txt", m, reducerId)
		file, err := os.Open(inputFile)
		if err != nil {
			log.Printf("Skipping missing file: %s\n", inputFile)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) != 2 {
				continue 
			}

			word := parts[0]
			// count, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Printf("Invalid file for word %s: %v\n", word, err)
				continue
			}
			if !contains(wordToFile[word], parts[1]) {
				wordToFile[word] = append(wordToFile[word], parts[1])
			}
		}
		file.Close()
	}

	outputFile, err := os.Create(fmt.Sprintf("reduce_output/reduce-%d.txt", reducerId))
	if err != nil {
		log.Fatalf("Failed to create reduce output file: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for word, files := range wordToFile {
		fmt.Fprintf(writer, "%s ", word)
		fmt.Fprintf(writer, "[")
		for _, filename := range files{
			fmt.Fprintf(writer, "%s, ", filename)
		}
		fmt.Fprintf(writer, "]\n")
	}
	writer.Flush()

	log.Printf("Reducer %d completed: Output stored in reduce-%d.txt\n", reducerId, reducerId)
}

func mergeAndReduceWordCounts(numMappers int, reducerId int) {
	wordCounts := make(map[string]int) 

	for m := range numMappers {
		inputFile := fmt.Sprintf("mapper_output/map-%d-%d.txt", m, reducerId)
		file, err := os.Open(inputFile)
		if err != nil {
			log.Printf("Skipping missing file: %s\n", inputFile)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) != 2 {
				continue 
			}

			word := parts[0]
			count, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Printf("Invalid count for word %s: %v\n", word, err)
				continue
			}
			wordCounts[word] += count
		}
		file.Close()
	}

	outputFile, err := os.Create(fmt.Sprintf("reduce_output/reduce-%d.txt", reducerId))
	if err != nil {
		log.Fatalf("Failed to create reduce output file: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for word, count := range wordCounts {
		fmt.Fprintf(writer, "%s %d\n", word, count)
	}
	writer.Flush()

	log.Printf("Reducer %d completed: Output stored in reduce-%d.txt\n", reducerId, reducerId)
	
}

func processReduceTask(numMappers int, reducerId int, taskType TaskType) {
	if taskType == COUNT_FREQUENCY{
		mergeAndReduceWordCounts(numMappers, reducerId)
	} else{
		mergeAndReduceInvertedIndex(numMappers, reducerId)
	}
	
	conn, err := grpc.NewClient(masterServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error while connecting to Master :%v", err)
	}
	masterclient := mapReducepb.NewSubmitResultServiceClient(conn)
	err = sendReduceResults(masterclient)
	if err != nil {
		log.Fatalf("Error while sending Map Results :%v", err)
	}

}


func getReducerIndex(word string, numReduce int) int {
	hash := fnv.New32a()
	hash.Write([]byte(word))
	return int(hash.Sum32()) % numReduce
}

func mapWordFrequency(mapperID int, inputFile string, numReduce int) {
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening input file:", err)
	}
	defer file.Close()

	intermediateFiles := make([]*os.File, numReduce)
	for i := range numReduce {
		filename := fmt.Sprintf("mapper_output/map-%d-%d.txt", mapperID, i) 
		intermediateFiles[i], _ = os.Create(filename)
		defer intermediateFiles[i].Close()
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text()) 
		for _, word := range words {
			reducerIndex := getReducerIndex(word, numReduce)
			fmt.Fprintf(intermediateFiles[reducerIndex], "%s 1\n", word) 
		}
	}
}

func mapInvertedIndex(mapperID int, inputFile string, numReduce int) {
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening input file:", err)
	}
	defer file.Close()

	intermediateFiles := make([]*os.File, numReduce)
	for i := range numReduce {
		filename := fmt.Sprintf("mapper_output/map-%d-%d.txt", mapperID, i) 
		intermediateFiles[i], _ = os.Create(filename)
		defer intermediateFiles[i].Close()
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text()) 
		for _, word := range words {
			reducerIndex := getReducerIndex(word, numReduce)
			fmt.Fprintf(intermediateFiles[reducerIndex], "%s %s\n", word, inputFile) 
		}
	}
}

func processMapTask(mapperID int, inputFile string, numReduce int, taskType TaskType) {
	if taskType == COUNT_FREQUENCY{
		mapWordFrequency(mapperID, inputFile, numReduce)
	} else{
		mapInvertedIndex(mapperID, inputFile, numReduce)
	}
	conn, err := grpc.NewClient(masterServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error while connecting to Master :%v", err)
	}
	masterclient := mapReducepb.NewSubmitResultServiceClient(conn)
	err = sendMapResults(masterclient)
	if err != nil {
		log.Fatalf("Error while sending Map Results :%v", err)
	}
}