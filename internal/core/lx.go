package core

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type logMessage struct {
	timestamp time.Time
	stream    string
	message   string
}

func Extract(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: go run main.go <command> [args...]")
		return
	}

	filteredText := args[0]
	command := args[1]
	commandArgs := args[2:]

	cmd := exec.Command(command, commandArgs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stderr pipe: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
	}

	logChannel := make(chan logMessage, 100)
	var wg sync.WaitGroup

	wg.Add(2)

	go streamReader("stdout", stdoutPipe, logChannel, &wg)
	go streamReader("stderr", stderrPipe, logChannel, &wg)

	go func() {
		wg.Wait()
		close(logChannel)
	}()

	for log := range logChannel {
		if strings.Contains(log.message, filteredText) {
			fmt.Printf("[%s][%s]: %s\n", log.timestamp.Format(time.RFC3339), log.stream, log.message)
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
	}

	fmt.Println("Command executed successfully.")
}

func streamReader(streamName string, reader io.ReadCloser, logChannel chan<- logMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logChannel <- logMessage{
			timestamp: time.Now(),
			stream:    streamName,
			message:   scanner.Text(),
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from %s: %v\n", streamName, err)
	}
}
