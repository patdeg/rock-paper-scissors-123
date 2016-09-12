// Rock Paper Scissors Game on App Engine
package main

import (
	"fmt"
	bigquery "google.golang.org/api/bigquery/v2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"net/http"
	"strings"
)

// Create BigQuery tables for plays and games in current project (admin only)
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
