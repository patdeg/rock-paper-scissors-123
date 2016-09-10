// Rock Paper Scissors Game on App Engine
package main

import (
	"deglonconsulting.com/common"
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"html/template"
	"math/rand"
	"net/http"
	"time"
)

var pageTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

var answers = []string{
	"rock",
	"paper",
	"scissor",
}

type GamePlay struct {
	DatastoreKey      string    `json:"datastore_key" datastore:"-"`
	CurrentUserPlay   string    `json:"current_user_play"`
	CurrentServerPlay string    `json:"current_server_play"`
	Last3UserPlays    string    `json:"last_3_user_play"`
	Last3ServerPlays  string    `json:"last_3_server_play"`
	Last2UserPlays    string    `json:"last_2_user_play"`
	Last2ServerPlays  string    `json:"last_2_server_play"`
	CreatedTime       time.Time `json:"created_time,omitempty"`
	UserEmail         string    `json:"user_email,omitempty"`
}

// HTML Template for the home page
var homeTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

// Main init function to assign paths to handlers
func init() {

	// Home page (& catch-all)
	http.HandleFunc("/", GameHandler)

	// API to get next server play
	http.HandleFunc("/play", PlayHandler)

	// API to record previous play
	http.HandleFunc("/record", RecordHandler)

}

func GameHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Game Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	log.Debugf(c, "User mode")
	if err := pageTemplate.Execute(w, template.FuncMap{
		"Path":    r.URL.Path,
		"City":    common.CamelCase(r.Header.Get("X-AppEngine-City")),
		"Version": appengine.VersionID(c),
	}); err != nil {
		log.Errorf(c, "Error with pageTemplate: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

}

func PlayHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Play Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	rand.Seed(time.Now().UnixNano())
	defaultValue := answers[rand.Intn(len(answers))]

	last3UserPlays := Simplify(c, r.FormValue("pu"))
	last3ServerPlays := Simplify(c, r.FormValue("ps"))
	last2UserPlays := LastTwo(last3UserPlays)
	last2ServerPlays := LastTwo(last3ServerPlays)
	log.Debugf(c, "%v / %v / %v / %v", last3UserPlays, last3ServerPlays, last2UserPlays, last2ServerPlays)

	var gamePlays []GamePlay
	q := datastore.NewQuery("GamePlay").
		Filter("Last2UserPlays =", last2UserPlays).
		Filter("Last2ServerPlays =", last2ServerPlays).
		Limit(100)
	_, err := q.GetAll(c, &gamePlays)
	if err != nil {
		log.Errorf(c, "Error, searching for previous plays: %v", err)
		log.Infof(c, "Providing default value")
		fmt.Fprint(w, defaultValue)
		return
	}

	if len(gamePlays) == 0 {
		log.Infof(c, "No statistics, providing default value")
		fmt.Fprint(w, defaultValue)
		return
	}

	freq := make(map[string]int)

	for _, gp := range gamePlays {
		freq[gp.CurrentUserPlay]++
	}

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

	answer := defaultValue
	switch {
	default:
		log.Errorf(c, "Unkown most frequent answer %v, showing default value", mostFreqPlay)
	case mostFreqPlay == "r":
		answer = "paper"
		log.Infof(c, "Most frequent sign is Rock, showing Paper")
	case mostFreqPlay == "p":
		answer = "scissor"
		log.Infof(c, "Most frequent sign is Paper, showing Scissor")
	case mostFreqPlay == "s":
		answer = "rock"
		log.Infof(c, "Most frequent sign is Scissor, showing Rock")
	}

	fmt.Fprint(w, answer)

}

func RecordHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Record Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	currentUserPlay := Simplify(c, r.FormValue("u"))
	if currentUserPlay == "" {
		log.Errorf(c, "Error, missing parameter u")
		http.Error(w, "Error, missing parameter", http.StatusInternalServerError)
		return
	}

	currentServerPlay := Simplify(c, r.FormValue("s"))
	if currentServerPlay == "" {
		log.Errorf(c, "Error, missing parameter s")
		http.Error(w, "Error, missing parameter", http.StatusInternalServerError)
		return
	}

	last3UserPlays := Simplify(c, r.FormValue("pu"))

	last3ServerPlays := Simplify(c, r.FormValue("ps"))

	gamePlay := GamePlay{
		CurrentUserPlay:   currentUserPlay,
		CurrentServerPlay: currentServerPlay,
		Last3UserPlays:    last3UserPlays,
		Last3ServerPlays:  last3ServerPlays,
		Last2UserPlays:    LastTwo(last3UserPlays),
		Last2ServerPlays:  LastTwo(last3ServerPlays),
		CreatedTime:       time.Now(),
		UserEmail:         user.Current(c).Email,
	}

	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "GamePlay", nil), &gamePlay); err != nil {
		log.Errorf(c, "Error while storing play: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
