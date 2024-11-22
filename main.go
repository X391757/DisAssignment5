package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Bidder struct {
	Name   string
	Amount int
}

type Auction struct {
	mutex         sync.RWMutex
	startTime     time.Time
	duration      time.Duration
	highestBid    int
	highestBidder string
	bidders       map[string]int
	status        string // "ongoing" or "ended"
}

var (
	auction = Auction{
		startTime:     time.Now(),
		duration:      100 * time.Second,
		highestBid:    0,
		highestBidder: "",
		bidders:       make(map[string]int),
		status:        "ongoing",
	}
	logger *log.Logger
)

func handleBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var body map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		logger.Printf("Failed to decode bid request: %v\n", err)
		return
	}

	// Validate input
	name, ok := body["name"].(string)
	amount, ok2 := body["amount"].(float64)
	if !ok || !ok2 {
		http.Error(w, "Invalid name or amount", http.StatusBadRequest)
		logger.Println("Invalid bid: missing or invalid name/amount")
		return
	}

	auction.mutex.Lock()
	defer auction.mutex.Unlock()

	// Check if auction has ended
	if auction.status == "ended" {
		response := map[string]string{"outcome": "auction ended"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		logger.Printf("Fail: auction ended for %s with amount %f\n", name, amount)
		return
	}

	// Register bidder if not already registered
	if _, exists := auction.bidders[name]; !exists {
		auction.bidders[name] = 0
	}

	// Validate that the bid is strictly higher than the current highest bid
	if int(amount) <= auction.highestBid {
		response := map[string]string{"outcome": "fail", "reason": "bid must be higher than current highest bid"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		logger.Printf("Fail: %s's bid %f is not higher than the current highest bid %d\n", name, amount, auction.highestBid)
		return
	}

	// Update the highest bid and highest bidder
	auction.highestBid = int(amount)
	auction.highestBidder = name

	// Update the bidder's last bid
	auction.bidders[name] = int(amount)

	response := map[string]string{"outcome": "success"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	logger.Printf("Success : %s placed a bid of %d\n", name, int(amount))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	auction.mutex.RLock()
	defer auction.mutex.RUnlock()

	response := map[string]interface{}{
		"status":         auction.status,
		"highest_bid":    auction.highestBid,
		"highest_bidder": auction.highestBidder,
		"time_remaining": int(auction.duration.Seconds()) - int(time.Since(auction.startTime).Seconds()),
	}

	if auction.status == "ended" {
		response["winner"] = auction.highestBidder
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	logger.Printf("Success : status=%s, highest_bid=%d, highest_bidder=%s\n", auction.status, auction.highestBid, auction.highestBidder)
}

func endAuction() {
	time.Sleep(auction.duration)
	auction.mutex.Lock()
	defer auction.mutex.Unlock()
	auction.status = "ended"
	logger.Printf("Auction ended! Winner: %s with bid %d\n", auction.highestBidder, auction.highestBid)
}

func main() {
	// Set up logging to a file
	logFile, err := os.OpenFile("auction.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	logger = log.New(logFile, "AUCTION: ", log.Ldate|log.Ltime|log.Lshortfile)

	port := flag.Int("port", 8080, "Port to run the auction server on")
	flag.Parse()

	// Start the auction end timer
	go endAuction()

	// Set up routes
	http.HandleFunc("/bid", handleBid)
	http.HandleFunc("/query", handleQuery)

	// Start the server
	logger.Printf("Starting Auction server on port %d\n", *port)
	fmt.Printf("Starting Auction server on port %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
