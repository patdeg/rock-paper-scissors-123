package main

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"net/http"
	"strings"
)

// Function for Handlers to check if user is logged in
// and redirect to login URL if required. Return TRUE when a redirect
// is needed (or a fatal error occurs when getting the Login URL).
// When returning TRUE, the Handler should just exit as the user will
// be redirected to the Login URL.
func RedirectIfNotLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	c := appengine.NewContext(r)
	if user.Current(c) == nil {
		redirectURL, err := user.LoginURL(c, r.URL.Path)
		if err != nil {
			log.Errorf(c, "Error getting LoginURL: %v", err)
			http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return true
	}
	return false
}

func Simplify(c context.Context, text string) string {
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

func LastTwo(text string) string {
	n := len(text) - 2
	if n < 0 {
		n = 0
	}
	return text[n:]
}
