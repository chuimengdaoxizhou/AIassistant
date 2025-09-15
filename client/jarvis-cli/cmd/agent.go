package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with the agent service",
}

var submitCmd = &cobra.Command{
	Use:   "submit [task description]",
	Short: "Submit a new task to the agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskDescription := args[0]
		submitTask(taskDescription)
	},
}

var watchCmd = &cobra.Command{
	Use:   "watch [task-id]",
	Short: "Watch for real-time results of a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskID := args[0]
		watchTask(taskID)
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(submitCmd)
	agentCmd.AddCommand(watchCmd)
}

func submitTask(description string) {
	apiURL := "http://localhost:8081/api/v1/tasks"

	payload := map[string]string{"content": description}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error creating JSON payload: %v", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Fatalf("Error submitting task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("Failed to submit task, status code: %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Error decoding response: %v", err)
	}

	fmt.Printf("Task submitted successfully!\nTask ID: %s\n", result["task_id"])
	fmt.Printf("To watch for results, run: jarvis-cli agent watch %s\n", result["task_id"])
}

func watchTask(taskID string) {
	// For now, we assume the websocket is on the same host.
	u := url.URL{Scheme: "ws", Host: "localhost:8081", Path: "/ws/subscribe"}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	fmt.Println("WebSocket connected. Waiting for results...")

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		
		// Pretty print the JSON output
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, message, "", "  "); err != nil {
			log.Printf("Error formatting JSON: %v. Raw message: %s", err, message)
		} else {
			fmt.Println(string(prettyJSON.Bytes()))
		}
	}
}
