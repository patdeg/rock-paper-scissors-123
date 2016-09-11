// Rock Paper Scissors Game on App Engine
package main

import (
	"deglonconsulting.com/common"
	"fmt"
	"github.com/mssola/user_agent"
	bigquery "google.golang.org/api/bigquery/v2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"html/template"
	"math/rand"
	"net/http"
	"strings"
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
	LastUserPlays     string    `json:"last_user_play"`
	LastServerPlays   string    `json:"last_server_play"`
	Last3UserPlays    string    `json:"last_3_user_play"`
	Last3ServerPlays  string    `json:"last_3_server_play"`
	Last2UserPlays    string    `json:"last_2_user_play"`
	Last2ServerPlays  string    `json:"last_2_server_play"`
	CreatedTime       time.Time `json:"created_time,omitempty"`
	CookieId          string    `json:"cookie_id,omitempty"`
}

type Request struct {
	Winner string `json:"winner,omitempty"`
	User   string `json:"user,omitempty"`
	Server string `json:"server,omitempty"`
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
	http.HandleFunc("/record", RecordPlayHandler)

	// API to record finished game
	http.HandleFunc("/game", RecordGameHandler)

	// Create Table in BigQuery (admin only)
	http.HandleFunc("/init", CreateBigQueryTableHandler)

}

func GameHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Game Handler")

	cookieId := GetCookieID(w, r)

	log.Debugf(c, "User mode")
	if err := pageTemplate.Execute(w, template.FuncMap{
		"Path":     r.URL.Path,
		"City":     common.CamelCase(r.Header.Get("X-AppEngine-City")),
		"Version":  appengine.VersionID(c),
		"CookieID": cookieId,
	}); err != nil {
		log.Errorf(c, "Error with pageTemplate: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

}

func PlayHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Play Handler")

	rand.Seed(time.Now().UnixNano())
	defaultValue := answers[rand.Intn(len(answers))]

	var gamePlays []GamePlay
	q := datastore.NewQuery("GamePlay").
		Filter("Last2UserPlays =", LastNCharacters(r.FormValue("pu"), 2)).
		Filter("Last2ServerPlays =", LastNCharacters(r.FormValue("pu"), 2)).
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

func RecordPlayHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Record Play Handler")

	cookieId := r.FormValue("id")

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

	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	ua := user_agent.New(r.Header.Get("User-Agent"))
	engineName, engineversion := ua.Engine()
	browserName, browserVersion := ua.Browser()

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

func RecordGameHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Record Game Handler")

	cookieId := r.FormValue("id")

	var data Request

	err := UnmarshalRequest(c, r, &data)
	if err != nil {
		log.Errorf(c, "Error while reading body: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debugf(c, "Winner: %v", data.Winner)
	log.Debugf(c, "User: %v", data.User)
	log.Debugf(c, "Server: %v", data.Server)

	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	ua := user_agent.New(r.Header.Get("User-Agent"))
	engineName, engineversion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	bq_req := &bigquery.TableDataInsertAllRequest{
		Kind: "bigquery#tableDataInsertAllRequest",
		Rows: []*bigquery.TableDataInsertAllRequestRows{
			{
				Json: map[string]bigquery.JsonValue{
					"CookieId":       cookieId,
					"User":           data.User,
					"Server":         data.Server,
					"Winner":         data.Winner,
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

// Create BigQuery table in current project (admin only)
func CreateBigQueryTableHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, ">>> Create BigQuery Table Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	if user.IsAdmin(c) == false {
		log.Errorf(c, "Error, user %v is not authorized to create table in BigQuery", user.Current(c).Email)
		http.Error(w, "Unauthorized Access", http.StatusUnauthorized)
		return
	}

	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	newTable := &bigquery.Table{
		TableReference: &bigquery.TableReference{
			ProjectId: projectId,
			DatasetId: "demo",
			TableId:   "plays",
		},
		FriendlyName: "Rock Paper Scissors Data",
		Schema: &bigquery.TableSchema{
			Fields: []*bigquery.TableFieldSchema{
				{Name: "CookieId", Type: "STRING", Description: "User Cookie Id"},
				{Name: "Time", Type: "TIMESTAMP", Description: "Time"},
				{Name: "User", Type: "STRING", Description: "Current User Play"},
				{Name: "Server", Type: "STRING", Description: "Current Server Play"},
				{Name: "LastUser", Type: "STRING", Description: "Last Previous User's Plays"},
				{Name: "LastServer", Type: "STRING", Description: "Last Previous Server's Plays"},
				{Name: "Country", Type: "STRING", Description: "Country"},
				{Name: "Region", Type: "STRING", Description: "Region"},
				{Name: "City", Type: "STRING", Description: "City"},
				{Name: "IsMobile", Type: "BOOLEAN", Description: "IsMobile"},
				{Name: "MozillaVersion", Type: "STRING", Description: "MozillaVersion"},
				{Name: "Platform", Type: "STRING", Description: "Platform"},
				{Name: "OS", Type: "STRING", Description: "OS"},
				{Name: "EngineName", Type: "STRING", Description: "EngineName"},
				{Name: "EngineVersion", Type: "STRING", Description: "EngineVersion"},
				{Name: "BrowserName", Type: "STRING", Description: "BrowserName"},
				{Name: "BrowserVersion", Type: "STRING", Description: "BrowserVersion"},
			},
		},
	}

	err := CreateTableInBigQuery(c, newTable)
	if err != nil {
		log.Errorf(c, "Error requesting table creation in BigQuery: %v", err)
		http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	newTable2 := &bigquery.Table{
		TableReference: &bigquery.TableReference{
			ProjectId: projectId,
			DatasetId: "demo",
			TableId:   "games",
		},
		FriendlyName: "Rock Paper Scissors Game Results",
		Schema: &bigquery.TableSchema{
			Fields: []*bigquery.TableFieldSchema{
				{Name: "CookieId", Type: "STRING", Description: "User Cookie Id"},
				{Name: "Time", Type: "TIMESTAMP", Description: "Time"},
				{Name: "User", Type: "STRING", Description: "User Plays"},
				{Name: "Server", Type: "STRING", Description: "Server Plays"},
				{Name: "Winner", Type: "STRING", Description: "Game Winner"},
				{Name: "Country", Type: "STRING", Description: "Country"},
				{Name: "Region", Type: "STRING", Description: "Region"},
				{Name: "City", Type: "STRING", Description: "City"},
				{Name: "IsMobile", Type: "BOOLEAN", Description: "IsMobile"},
				{Name: "MozillaVersion", Type: "STRING", Description: "MozillaVersion"},
				{Name: "Platform", Type: "STRING", Description: "Platform"},
				{Name: "OS", Type: "STRING", Description: "OS"},
				{Name: "EngineName", Type: "STRING", Description: "EngineName"},
				{Name: "EngineVersion", Type: "STRING", Description: "EngineVersion"},
				{Name: "BrowserName", Type: "STRING", Description: "BrowserName"},
				{Name: "BrowserVersion", Type: "STRING", Description: "BrowserVersion"},
			},
		},
	}

	err = CreateTableInBigQuery(c, newTable2)
	if err != nil {
		log.Errorf(c, "Error requesting table creation in BigQuery: %v", err)
		http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "<h1>Table Created</h1>")
}
