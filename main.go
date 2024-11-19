package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

var (
	queries    = flag.String("queries", "Telejornal,News,Politics", "Comma-separated list of search terms for YouTube videos")
	maxResults = flag.Int64("max-results", 25, "Maximum number of YouTube results to retrieve")
)

const developerKey = "AIzaSyDlcqlvJPBTC-y71JruPLTCLTttCz0AXEg"

func main() {
	// Parse command-line flags
	flag.Parse()

	// Split the queries into a slice of strings
	queryList := strings.Split(*queries, ",")

	// Initialize the HTTP client with the API key
	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	// Create a new YouTube service
	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	// Create a file to store the results
	file, err := os.Create("youtube_ids.txt")
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	// Get the current time
	currentTime := time.Now()

	// Iterate over each query term
	for _, query := range queryList {
		// Print the query term to the file
		fmt.Fprintf(file, "Search results for: %v\n", query)

		// Make the API call to YouTube
		call := service.Search.List([]string{"id", "snippet"}).
			Q(query).
			MaxResults(*maxResults)
		response, err := call.Do()
		if err != nil {
			log.Printf("Error making API call to YouTube for query '%v': %v", query, err)
			continue
		}

		// Group video, channel, and playlist results in separate lists
		videos := make(map[string]string)
		channels := make(map[string]string)
		playlists := make(map[string]string)

		// Iterate through each item and check if the video was published in the last 24 hours
		for _, item := range response.Items {
			// Parse the publishedAt timestamp
			publishedAt, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			if err != nil {
				log.Printf("Error parsing published date for item %v: %v", item.Snippet.Title, err)
				continue
			}

			// If the video was published within the last 24 hours, add it to the appropriate list
			if currentTime.Sub(publishedAt).Hours() <= 24 {
				switch item.Id.Kind {
				case "youtube#video":
					videos[item.Id.VideoId] = item.Snippet.Title
				case "youtube#channel":
					channels[item.Id.ChannelId] = item.Snippet.Title
				case "youtube#playlist":
					playlists[item.Id.PlaylistId] = item.Snippet.Title
				}
			}
		}

		// Write the results for this query to the file
		writeIDs(file, "Videos", videos)
		writeIDs(file, "Channels", channels)
		writeIDs(file, "Playlists", playlists)

		// Add a newline separator between different query results
		fmt.Fprintln(file)
	}

	fmt.Println("Resultados salvos em um ficheiro youtube_ids.txt")
}

// writeIDs writes the ID and title of each result in a list, along with a section name, to the file
func writeIDs(file *os.File, sectionName string, matches map[string]string) {
	fmt.Fprintf(file, "%v:\n", sectionName)
	for id, title := range matches {
		fmt.Fprintf(file, "[%v] %v\n", id, title)
	}
	fmt.Fprintln(file)
}
