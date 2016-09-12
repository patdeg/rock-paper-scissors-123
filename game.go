// Rock Paper Scissors Game on App Engine
package main

import (
	"fmt"
	"github.com/mssola/user_agent"
	bigquery "google.golang.org/api/bigquery/v2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Three possible answers from the game
var answers = []string{
	"rock",
	"paper",
	"scissor",
}

// Structure to store a game play in Datastore
type GamePlay struct {
	CurrentUserPlay   string    `json:"current_user_play"`
	CurrentServerPlay string    `json:"current_server_play"`
	LastUserPlays     string    `json:"last_user_play"`
	LastServerPlays   string    `json:"last_server_play"`
	Last3UserPlays    string    `json:"last_3_user_play"`
	Last3ServerPlays  string    `json:"last_3_server_play"`
	Last2UserPlays    string    `json:"last_2_user_play"`
	Last2ServerPlays  string    `json:"last_2_server_play"`
	CreatedTime       time.Time `json:"created_time,omitempty"`
	CookieId          string    `json:"cookie_id,omitempty"`
}

// Structure used to record a game at its end
type Request struct {
	Winner string `json:"winner,omitempty"`
	User   string `json:"user,omitempty"`
	Server string `json:"server,omitempty"`
}

// Compress a set of plays by their first letter.
// For example "rock paper rock" will return "rpr"
func Compress(text string) string {
	result := ""
	for _, w := range strings.Split(text, " ") {
		for _, a := range answers {
			if w == a {
				result += string(a[0])
				break
			}
		}
	}
	return result
}

// Handler to provide the server next play/move
// Return rock, paper or scissors in HTTP response
func PlayHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Play Handler")

	// Shuffle random generator with Unix time
	rand.Seed(time.Now().UnixNano())

	// Pick randomly one of the play/move.
	// To be used as default value for this request
	defaultValue := answers[rand.Intn(len(answers))]

	// Get at most 100 previous plays/moves with same conditions
	// in the last 2 round for users and server
	var gamePlays []GamePlay
	q := datastore.NewQuery("GamePlay").
		Filter("Last2UserPlays =", LastNCharacters(r.FormValue("pu"), 2)).
		Filter("Last2ServerPlays =", LastNCharacters(r.FormValue("pu"), 2)).
		Limit(100)
	_, err := q.GetAll(c, &gamePlays)

	// If error, return default (random) value after emiting error message in log
	if err != nil {
		log.Errorf(c, "Error, searching for previous plays: %v", err)
		log.Infof(c, "Providing default value")
		fmt.Fprint(w, defaultValue)
		return
	}

	// If no plays/moves in datastore, return default (random) value
	if len(gamePlays) == 0 {
		log.Infof(c, "No statistics, providing default value")
		fmt.Fprint(w, defaultValue)
		return
	}

	// Create frequency histogram of user's move/play
	freq := make(map[string]int)
	for _, gp := range gamePlays {
		freq[gp.CurrentUserPlay]++
	}

	// Find the most common play/move
	//TODO: improve randomness in case of equality between 2 or 3 plays
	mostFreqPlay := ""
	for p, _ := range freq {
		if mostFreqPlay == "" {
			mostFreqPlay = p
		} else {
			if freq[p] > freq[mostFreqPlay] {
				mostFreqPlay = p
			}
		}
	}

	// Uncompress play (r,p,s to rock, paper, scissors), and
	// provide opposite play (i.e. paper for rock, rock for scissors, or scissors for paper)
	answer := defaultValue
	switch {
	default:
		log.Errorf(c, "Unkown most frequent answer %v, showing default value", mostFreqPlay)
	case mostFreqPlay == "r":
		answer = "paper"
		log.Debugf(c, "Most frequent sign is Rock, showing Paper")
	case mostFreqPlay == "p":
		answer = "scissor"
		log.Debugf(c, "Most frequent sign is Paper, showing Scissor")
	case mostFreqPlay == "s":
		answer = "rock"
		log.Debugf(c, "Most frequent sign is Scissor, showing Rock")
	}

	// Return final answer to HTTP response
	fmt.Fprint(w, answer)

}

// Record a play/move once it has been played
func RecordPlayHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Record Play Handler")

	// Get User Cookie Id
	cookieId := r.FormValue("id")

	// Get and compress current user play, emmits error if empty
	currentUserPlay := Compress(r.FormValue("u"))
	if currentUserPlay == "" {
		log.Errorf(c, "Error, missing parameter u")
		http.Error(w, "Error, missing parameter", http.StatusInternalServerError)
		return
	}

	// Get and compress current server play, emmits error if empty
	currentServerPlay := Compress(r.FormValue("s"))
	if currentServerPlay == "" {
		log.Errorf(c, "Error, missing parameter s")
		http.Error(w, "Error, missing parameter", http.StatusInternalServerError)
		return
	}

	// Record play in Datastore
	gamePlay := GamePlay{
		CurrentUserPlay:   currentUserPlay,
		CurrentServerPlay: currentServerPlay,
		LastUserPlays:     r.FormValue("pu"),
		LastServerPlays:   r.FormValue("ps"),
		Last3UserPlays:    LastNCharacters(r.FormValue("pu"), 3),
		Last3ServerPlays:  LastNCharacters(r.FormValue("ps"), 3),
		Last2UserPlays:    LastNCharacters(r.FormValue("pu"), 2),
		Last2ServerPlays:  LastNCharacters(r.FormValue("ps"), 2),
		CreatedTime:       time.Now(),
		CookieId:          cookieId,
	}
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "GamePlay", nil), &gamePlay); err != nil {
		log.Errorf(c, "Error while storing play: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get project Id where to store data in BigQuery
	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	// Extract some basic information of User Agent
	ua := user_agent.New(r.Header.Get("User-Agent"))
	engineName, engineversion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	// Store play in Big Query
	bq_req := &bigquery.TableDataInsertAllRequest{
		Kind: "bigquery#tableDataInsertAllRequest",
		Rows: []*bigquery.TableDataInsertAllRequestRows{
			{
				Json: map[string]bigquery.JsonValue{
					"CookieId":       cookieId,
					"Time":           time.Now(),
					"User":           currentUserPlay,
					"Server":         currentServerPlay,
					"LastUser":       r.FormValue("pu"),
					"LastServer":     r.FormValue("ps"),
					"Country":        r.Header.Get("X-AppEngine-Country"),
					"Region":         r.Header.Get("X-AppEngine-Region"),
					"City":           r.Header.Get("X-AppEngine-City"),
					"IsMobile":       ua.Mobile(),
					"MozillaVersion": ua.Mozilla(),
					"Platform":       ua.Platform(),
					"OS":             ua.OS(),
					"EngineName":     engineName,
					"EngineVersion":  engineversion,
					"BrowserName":    browserName,
					"BrowserVersion": browserVersion,
				},
			},
		},
	}
	err := StreamDataInBigquery(c, projectId, "demo", "plays", bq_req)
	if err != nil {
		log.Errorf(c, "Error while streaming visit to BigQuery: %v", err)
		log.Debugf(c, "Request: %v", ToJSON(bq_req))
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

}

// Record game when it is finished
func RecordGameHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Record Game Handler")

	// Get Cookie Id
	cookieId := r.FormValue("id")

	// Extract game information from Request's body
	var gameInfo Request
	err := UnmarshalRequest(c, r, &gameInfo)
	if err != nil {
		log.Errorf(c, "Error while reading body: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get project Id where to store data in BigQuery
	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	// Extract some basic information of User Agent
	ua := user_agent.New(r.Header.Get("User-Agent"))
	engineName, engineversion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	// Store game in Big Query
	bq_req := &bigquery.TableDataInsertAllRequest{
		Kind: "bigquery#tableDataInsertAllRequest",
		Rows: []*bigquery.TableDataInsertAllRequestRows{
			{
				Json: map[string]bigquery.JsonValue{
					"CookieId":       cookieId,
					"User":           gameInfo.User,
					"Server":         gameInfo.Server,
					"Winner":         gameInfo.Winner,
					"Time":           time.Now(),
					"Country":        r.Header.Get("X-AppEngine-Country"),
					"Region":         r.Header.Get("X-AppEngine-Region"),
					"City":           r.Header.Get("X-AppEngine-City"),
					"IsMobile":       ua.Mobile(),
					"MozillaVersion": ua.Mozilla(),
					"Platform":       ua.Platform(),
					"OS":             ua.OS(),
					"EngineName":     engineName,
					"EngineVersion":  engineversion,
					"BrowserName":    browserName,
					"BrowserVersion": browserVersion,
				},
			},
		},
	}
	err = StreamDataInBigquery(c, projectId, "demo", "games", bq_req)
	if err != nil {
		log.Errorf(c, "Error while streaming visit to BigQuery: %v", err)
		log.Debugf(c, "Request: %v", ToJSON(bq_req))
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

}
