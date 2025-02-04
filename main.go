// main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// LogMessage holds the details of an incoming log entry.
type LogMessage struct {
	Sender string
	Level  string
	Text   string
}

var (
	telegramBotToken string
	telegramChatID   string
	mergeInterval    time.Duration
)

// sendTelegramMessage sends a message to Telegram using the Bot API.
// It sends the message as Markdown-formatted text.
func sendTelegramMessage(message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	payload := map[string]interface{}{
		"chat_id":    telegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Telegram returns HTTP 200 on success.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-OK response: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// aggregator collects LogMessages from the msgChan channel and, after waiting
// for the merge interval, joins them into one message which is then sent to Telegram.
func aggregator(msgChan <-chan LogMessage) {
	for {
		// Block until at least one message is available.
		msg := <-msgChan
		messages := []LogMessage{msg}

		// Start a timer for the merge interval.
		timer := time.NewTimer(mergeInterval)
	loop:
		for {
			select {
			case m := <-msgChan:
				messages = append(messages, m)
			case <-timer.C:
				break loop
			}
		}

		// Combine messages. For each message, if a sender is provided, prefix the text.
		combined := ""
		for _, m := range messages {
			if m.Sender != "" {
				combined += fmt.Sprintf("From: *%s*: ", m.Sender)
			}
			if m.Level != "" {
				combined += fmt.Sprintf("`[%s]` ", m.Level)
			}
			combined += m.Text + "\n"
		}

		// Send the aggregated message to Telegram.
		if err := sendTelegramMessage(combined); err != nil {
			log.Printf("Error sending message to Telegram: %v", err)
			// Attempt to notify via Telegram that an error occurred.
			errorMsg := fmt.Sprintf("Error occurred while processing log message: %v", err)
			if err2 := sendTelegramMessage(errorMsg); err2 != nil {
				log.Printf("Error sending error notification to Telegram: %v", err2)
			}
		}
	}
}

// submitHandler handles POST requests to /notification. It reads the Markdown text from
// the request body and extracts the optional query parameters "sender" and "level".
// On error (e.g. empty body), it returns an error to the client and sends a Telegram
// notification.
func submitHandler(msgChan chan<- LogMessage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			errMsg := "Empty or invalid message body received"
			http.Error(w, errMsg, http.StatusBadRequest)
			// Notify via Telegram about the error.
			sendTelegramMessage(fmt.Sprintf("Error: %s", errMsg))
			return
		}
		text := string(body)

		// Retrieve optional query parameters.
		sender := r.URL.Query().Get("sender")
		level := r.URL.Query().Get("level")

		// Send the log message to the aggregator.
		msg := LogMessage{
			Sender: sender,
			Level:  level,
			Text:   text,
		}

		// Use the channel (with a buffer) to avoid blocking.
		select {
		case msgChan <- msg:
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("Message received"))
		default:
			http.Error(w, "Server busy", http.StatusServiceUnavailable)
		}
	}
}

func main() {
	// Load environment variables from the .env file if it exists.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using OS environment variables")
	}
	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	mergeIntervalStr := os.Getenv("MERGE_INTERVAL")
	if mergeIntervalStr == "" {
		mergeInterval = 1 * time.Second
	} else {
		var err error
		mergeInterval, err = time.ParseDuration(mergeIntervalStr)
		if err != nil {
			log.Printf("Invalid MERGE_INTERVAL, defaulting to 1s: %v", err)
			mergeInterval = 1 * time.Second
		}
	}

	// Ensure that required Telegram credentials are provided.
	if telegramBotToken == "" || telegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set")
	}

	// Create a buffered channel to receive log messages.
	msgChan := make(chan LogMessage, 100)

	// Start the aggregator goroutine.
	go aggregator(msgChan)

	// Register the HTTP handler.
	http.HandleFunc("/notification", submitHandler(msgChan))

	// Use PORT from env (default to 10000 if not set).
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	addr := ":" + port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
