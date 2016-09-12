// Rock Paper Scissors Game on App Engine
package main

import (
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"html/template"
	"net/http"
	"strings"
)

// HTML Template for the home page
var pageTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

// Root Handler for the application
func HomeHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)

	log.Infof(c, ">>>> Home Handler")

	// Check if game is a Facebook canvas
	log.Debugf(c, "Referer: %v")
	isFacebook := ""
	if strings.Contains(r.Referer(), "apps.facebook.com") && r.Method == "POST" {
		isFacebook = "1"
	}

	// Get existing ID in cookie, set it up if it doesn't exist
	cookieId := GetCookieID(w, r)

	// Render home page
	if err := pageTemplate.Execute(w, template.FuncMap{
		"Version":    appengine.VersionID(c),
		"CookieID":   cookieId,
		"isFacebook": isFacebook,
	}); err != nil {
		log.Errorf(c, "Error with pageTemplate: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

}
