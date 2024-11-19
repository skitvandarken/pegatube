package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const developerKey = "AIzaSyDlcqlvJPBTC-y71JruPLTCLTttCz0AXEg"

func main() {
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/search", handleSearch)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html") // Serve the front-end HTML
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query().Get("queries")
	maxResults := r.URL.Query().Get("max-results")

	client := &http.Client{Transport: &transport.APIKey{Key: developerKey}}
	service, err := youtube.New(client)
	if err != nil {
		http.Error(w, "Error creating YouTube client", http.StatusInternalServerError)
		return
	}

	terms := strings.Split(queries, ",")
	results := []map[string]string{}

	for _, term := range terms {
		call := service.Search.List([]string{"id", "snippet"}).
			Q(strings.TrimSpace(term)).
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func parseMaxResults(maxResults string) int64 {
	if n, err := strconv.ParseInt(maxResults, 10, 64); err == nil {
		return n
	}
	return 10
}
