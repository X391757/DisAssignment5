package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Coordinator struct {
	replicas []string // addr of replica
	logger   *log.Logger
}

func (c *Coordinator) bid(name string, amount int) {
	data := map[string]interface{}{"name": name, "amount": amount}
	body, _ := json.Marshal(data)

	for _, replica := range c.replicas {
		resp, err := http.Post(replica+"/bid", "application/json", bytes.NewBuffer(body))
		if err != nil {
			c.logger.Printf("Error sending bid to replica %s: %v\n", replica, err)
			continue
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			c.logger.Printf("Error decoding response from replica %s: %v\n", replica, err)
			continue
		}

		if reason, ok := response["reason"].(string); ok && reason == "fail" {
			c.logger.Printf("Replica %s returned failure: %s\n", replica, response)
		}
	}
	c.logger.Println("Bid request completed")
}

// bid
func (c *Coordinator) query() {
	for _, replica := range c.replicas {
		data, err := c.queryFromReplica(replica)
		if err != nil {
			c.logger.Printf("Error querying replica %s: %v\n", replica, err)
			continue
		}
		c.logger.Printf("Data from replica %s: %s\n", replica, data)
		return
	}
	c.logger.Println("Failed to query all replicas")
}

// query
func (c *Coordinator) queryFromReplica(replica string) (string, error) {
	client := &http.Client{
		Timeout: 2 * time.Second, // set timeout time
	}
	resp, err := client.Get(replica + "/query")
	if err != nil {
		return "", fmt.Errorf("failed to connect to replica: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return string(body), nil
}

func main() {
	// open log
	logFile, err := os.OpenFile("program.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	// init log
	logger := log.New(logFile, "LOG: ", log.Ldate|log.Ltime|log.Lshortfile)

	coordinator := &Coordinator{
		replicas: []string{
			"http://localhost:8080",
			"http://localhost:8081",
		},
		logger: logger,
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nEnter a command:")
		fmt.Println("1 <name> <amount>: Place a bid")
		fmt.Println("2: Query auction status")
		fmt.Println("0: Exit")
		fmt.Print("> ")

		// read input
		if !scanner.Scan() {
			logger.Println("Error reading input. Exiting.")
			break
		}
		input := strings.TrimSpace(scanner.Text())
		args := strings.Split(input, " ")

		if len(args) == 0 {
			fmt.Println("Invalid input. Try again.")
			continue
		}

		switch args[0] {
		case "1":
			// bid
			if len(args) != 3 {
				fmt.Println("Invalid input. Usage: 1 <name> <amount>")
				continue
			}

			name := args[1]
			amount, err := strconv.Atoi(args[2])
			if err != nil {
				fmt.Println("Invalid amount. Please enter a valid integer.")
				continue
			}

			coordinator.logger.Printf("Placing bid: Name=%s, Amount=%d\n", name, amount)
			coordinator.bid(name, amount)

		case "2":
			// query
			coordinator.logger.Println("Querying auction status...")
			coordinator.query()

		case "0":
			// exit
			coordinator.logger.Println("Exiting...")
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid command. Try again.")
		}
	}
}
