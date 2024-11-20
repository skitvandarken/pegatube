package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	// Retrieve the API key from the environment variable
	developerKey := os.Getenv("YOUTUBE_API_KEY")
	if developerKey == "" {
		log.Fatal("API key is missing. Make sure it is set in the .env file.")
	}

	// Test the API connection
	if !testAPIConnection(developerKey) {
		log.Fatal("Failed to connect to the YouTube API. Check your API key and network connection.")
	}

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, developerKey)
	})

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// testAPIConnection tests the API connection and returns false if the connection fails
func testAPIConnection(developerKey string) bool {
	client := &http.Client{Transport: &transport.APIKey{Key: developerKey}}
	service, err := youtube.New(client)
	if err != nil {
		return false
	}

	// Perform a simple search query to test the connection
	call := service.Search.List([]string{"id"}).
		Q("Telejornal"). // Arbitrary search term for testing
		MaxResults(1)

	_, err = call.Do()
	return err == nil
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html") // Serve the front-end HTML
}

func handleSearch(w http.ResponseWriter, r *http.Request, developerKey string) {
	queries := r.URL.Query().Get("queries")
	maxResults := r.URL.Query().Get("max-results")

	client := &http.Client{Transport: &transport.APIKey{Key: developerKey}}
	service, err := youtube.New(client)
	if err != nil {
		http.Error(w, "Error creating YouTube client", http.StatusInternalServerError)
		return
	}

	// Calculate the timestamp for 24 hours ago
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	// Define the specific channel IDs to search within
	channelIDs := []string{
		"UCpf7-LhTbmKk11p4nqw5LYA", // TPA Online
		"UCxRiylOpWTvJLamhlm63VNw", // TV Zimbo Oficial
	}

	terms := strings.Split(queries, ",")
	results := []map[string]string{}

	for _, term := range terms {
		for _, channelID := range channelIDs {
			call := service.Search.List([]string{"id", "snippet"}).
				Q(strings.TrimSpace(term)).
				ChannelId(channelID).               // Filter results by channel ID
				PublishedAfter(twentyFourHoursAgo). // Filter videos published after 24 hours ago
				MaxResults(parseMaxResults(maxResults))
			response, err := call.Do()
			if err != nil {
				http.Error(w, "Error making API call to YouTube", http.StatusInternalServerError)
				return
			}

			for _, item := range response.Items {
				if item.Id.Kind == "youtube#video" {
					results = append(results, map[string]string{
						"id":    item.Id.VideoId,
						"title": item.Snippet.Title,
					})
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func parseMaxResults(maxResults string) int64 {
	if n, err := strconv.ParseInt(maxResults, 10, 64); err == nil {
		return n
	}
	return 10
}
