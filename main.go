// Rock Paper Scissors Game on App Engine
package main

import (
	"net/http"
)

// Main init function to assign paths to handlers
func init() {

	// Home page (& catch-all)
	http.HandleFunc("/", HomeHandler)

	// API to get next server play
	http.HandleFunc("/play", PlayHandler)

	// API to record previous play
	http.HandleFunc("/record", RecordPlayHandler)

	// API to record finished game
	http.HandleFunc("/game", RecordGameHandler)

	// Create Table in BigQuery (admin only)
	http.HandleFunc("/init", CreateBigQueryTableHandler)

}
