package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	bigquery "google.golang.org/api/bigquery/v2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Errors from the BigQuery functions
var (
	ErrorUndefinedTable         = errors.New("No newTable defined for CreateTableInBigQuery")
	ErrorUndefiedTableReference = errors.New("No newTable.TableReference defined for CreateTableInBigQuery")
	ErrorUndefiedSchema         = errors.New("No newTable.Schema defined for CreateTableInBigQuery")
	ErrorNoRequest              = errors.New("No req defined for StreamDataInBigquery")
	ErrorWhileStreaming         = errors.New("There was an error streaming data to Big Query")
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

// Return last N characters from text,
// return text if less than N characters
func LastNCharacters(text string, n int) string {
	m := len(text) - n
	if m < 0 {
		m = 0
	}
	return text[m:]
}

// Small utility function to convert a byte to a string
func BytesToString(b []byte) (s string) {
	n := bytes.Index(b, []byte{0})
	if n > 0 {
		s = string(b[:n])
	} else {
		s = string(b)
	}
	return
}

// Function to convert an object into a JSON string
func ToJSON(d interface{}) string {
	jsonData, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return BytesToString(jsonData)
}

// Get BigQuery Service Account Client
func GetBQServiceAccountClient(c context.Context) (*bigquery.Service, error) {
	serviceAccountClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(c,
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/bigquery"),
			Base: &urlfetch.Transport{
				Context: c,
			},
		},
	}
	return bigquery.New(serviceAccountClient)
}

// Return true if error is "Already Exists" for table/dataset creation
func ErrorIsAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), "Already Exists: ")
}

// Create a Table in BigQuery
func CreateTableInBigQuery(c context.Context, newTable *bigquery.Table) error {

	// Check validity of request
	if newTable == nil {
		return ErrorUndefinedTable
	}

	if newTable.TableReference == nil {
		return ErrorUndefiedTableReference
	}

	if newTable.Schema == nil {
		return ErrorUndefiedSchema
	}

	// Get BigQuery Service Account Client
	bqServiceAccountService, err := GetBQServiceAccountClient(c)
	if err != nil {
		log.Errorf(c, "Error getting BigQuery Service: %v", err)
		return err
	}

	newDataset := &bigquery.Dataset{
		DatasetReference: &bigquery.DatasetReference{
			ProjectId: newTable.TableReference.ProjectId,
			DatasetId: newTable.TableReference.DatasetId,
		},
	}

	// Create New Dataset
	_, err = bigquery.
		NewDatasetsService(bqServiceAccountService).
		Insert(
		newTable.TableReference.ProjectId,
		newDataset).
		Do()
	if (err != nil) && !ErrorIsAlreadyExists(err) {
		log.Errorf(c, "There was an error while creating dataset: %v", err)
		return err
	}

	// Create New Table
	_, err = bigquery.
		NewTablesService(bqServiceAccountService).
		Insert(
		newTable.TableReference.ProjectId,
		newTable.TableReference.DatasetId,
		newTable).
		Do()
	if (err != nil) && !ErrorIsAlreadyExists(err) {
		log.Errorf(c, "There was an error while creating table: %v", err)
		return err
	}

	return nil
}

// Stream data to BigQuery
func StreamDataInBigquery(c context.Context, projectId, datasetId, tableId string, req *bigquery.TableDataInsertAllRequest) error {

	if req == nil {
		return ErrorNoRequest
	}

	bqServiceAccountService, err := GetBQServiceAccountClient(c)
	if err != nil {
		log.Errorf(c, "Error getting BigQuery Service: %v", err)
		return err
	}

	resp, err := bigquery.
		NewTabledataService(bqServiceAccountService).
		InsertAll(projectId, datasetId, tableId, req).
		Do()
	if err != nil {
		log.Warningf(c, "Error streaming data to Big Query, trying again in 10 seconds: %v", err)
		time.Sleep(time.Second * 10)
		resp, err = bigquery.
			NewTabledataService(bqServiceAccountService).
			InsertAll(projectId, datasetId, tableId, req).
			Do()
		if err != nil {
			log.Errorf(c, "Error again streaming data to Big Query: %v", err)
			return err
		} else {
			log.Debugf(c, "2nd try was successful")
		}
	}

	isError := false
	for i, insertError := range resp.InsertErrors {
		if insertError != nil {
			for j, e := range insertError.Errors {
				if (e.DebugInfo != "") || (e.Message != "") || (e.Reason != "") {
					log.Errorf(c, "BigQuery error %v: %v at %v/%v", e.Reason, e.Message, i, j)
					isError = true
				}
			}
		}
	}

	if isError {
		return ErrorWhileStreaming
	}

	return nil

}

// Return a MD5 hash of a string
func MD5(data string) string {
	h := md5.New()
	io.WriteString(h, data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Get Cookie Id from cookies. If it doesn't exist, create, store
// in cookie and return the value
func GetCookieID(w http.ResponseWriter, r *http.Request) string {
	var id string
	c := appengine.NewContext(r)
	cookie, err := r.Cookie("ID")
	log.Infof(c, "ID cookie: %v", cookie)
	if err != nil || cookie == nil || cookie.Value == "" {
		ts := strconv.FormatInt(time.Now().UnixNano(), 10)
		id = MD5(ts + r.RemoteAddr)
		http.SetCookie(w, &http.Cookie{
			Name:    "ID",
			Value:   id,
			Path:    "/",
			Domain:  r.Host,
			Expires: time.Now().Add(time.Hour * 24 * 30),
		})
		log.Infof(c, "New Cookie = %v", id)
	} else {
		id = cookie.Value
		log.Infof(c, "Existing ID Cookie = %v", id)
	}
	return id
}

// Check if Cookie Id is in user's cookies
func DoesCookieExists(r *http.Request) bool {
	cookie, err := r.Cookie("ID")
	if err != nil || cookie == nil || cookie.Value == "" {
		return false
	}
	return true
}

// Utitility to convert JSON object in body
func UnmarshalRequest(c context.Context, r *http.Request, value interface{}) error {
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(r.Body)
	err := json.Unmarshal(buffer.Bytes(), value)
	if err != nil {
		log.Errorf(c, "Error while decoing JSON: %v", err)
		log.Infof(c, "JSON: %v", buffer.String())
		return err
	}
	return nil
}
