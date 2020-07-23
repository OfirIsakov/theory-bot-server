package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const port = ":6969"
const maxQuestionCount = 30

var creds credentials

// Making another interface with a getStatus method so the error could have more then 1 method
type errorStatus interface {
	GetStatus() int
}

// Encapsulating the interfaces into 1
type webError interface {
	error
	errorStatus
}

// The error struct
type httpError struct {
	What   string
	Status int
}

// Credentials struct containing username and password to each page
type credentials struct {
	Username string
	Password string
}

type questionResponse struct {
	Question     string
	WrongAnswer1 string
	WrongAnswer2 string
	WrongAnswer3 string
	RightAnswer  string
	Image        string
}

func (e *httpError) Error() string {
	return fmt.Sprintln(e.What)
}

func (e *httpError) GetStatus() int {
	return e.Status
}

func validateRequest(endPoint string, req *http.Request) webError {
	if req.URL.Path != endPoint {
		return &httpError{
			"404 Page Not Found.",
			http.StatusNotFound,
		}
	}
	userAgent := req.Header.Get("User-Agent")
	if userAgent != "TheoryBot" {
		return &httpError{
			"Invalid User Agent!",
			http.StatusInternalServerError,
		}
	}
	return nil
}

func fillStructWithJSON(w http.ResponseWriter, req *http.Request) (map[string]interface{}, webError) {
	var resultJSON map[string]interface{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&resultJSON)
	if err != nil {
		return nil, &httpError{
			"Error While Decoding JSON! Did You Send It Wrong?",
			http.StatusInternalServerError,
		}
	}

	log.Println(resultJSON)

	return resultJSON, nil
}

func sendError(w http.ResponseWriter, err string, status int) {
	log.Println(err)
	http.Error(w, err, status)
}

func startGameHandler(w http.ResponseWriter, req *http.Request) {
	if err := validateRequest("/startGame", req); err != nil {
		sendError(w, err.Error(), err.GetStatus())
		return
	}

	result, err := fillStructWithJSON(w, req)
	if err != nil {
		sendError(w, err.Error(), err.GetStatus())
		return
	}

	userID, isOkUserID := result["UserID"].(string)
	messageID, isOkMessageID := result["MessageID"].(string)
	questionCountFloat64, isOkQuestionCount := result["QuestionCount"].(float64)
	questionCount := int8(questionCountFloat64)
	if isOkUserID || isOkMessageID || isOkQuestionCount {
		sendError(w, "Please specify UserID(string), MessageID(string) and QuestionCount(int) in the JSON!", http.StatusBadRequest)
		return
	}
	if !(maxQuestionCount >= questionCount && questionCount > 0) {
		sendError(w, "Invalid quesiton count! Please give a value from 0 to 30", http.StatusBadRequest)
		return
	}

	if req.Method != "POST" {
		sendError(w, "Method no support!", http.StatusBadRequest)
		return
	}

	//TODO: Handle the request instead of printing
	fmt.Fprintf(w, "User: %s, Game: %s, Questions: %d\n", userID, messageID, questionCount)
	log.Printf("User: %s, Game: %s, Questions: %d\n", userID, messageID, questionCount)
}

func getNextHandler(w http.ResponseWriter, req *http.Request) {
	//TODO
	// This will handle users in game, the bot will send a userID and the server will
	// respond with a question || with statistics || error ...
}

func bindEndPointsToRoutes(username, password string) {
	http.HandleFunc("/startGame", basicAuth(startGameHandler))
	http.HandleFunc("/getNext", basicAuth(getNextHandler))
}

func validateCredentials(username, password string) bool {
	return subtle.ConstantTimeCompare([]byte(creds.Username), []byte(username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(creds.Password), []byte(password)) == 1
}

func basicAuth(handler http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		username, password, ok := r.BasicAuth()

		if !ok || !validateCredentials(username, password) {
			w.Header().Set("WWW-Authenticate", "Basic")
			sendError(w, "Unauthorized, invalid credentials", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

func main() {
	// Open the file
	jsonFile, err := os.Open("creds.json")
	if err != nil {
		log.Println("Error while opening the credentials file! Does it exsit?")
		log.Fatal(err)
	}

	// Load the file into a byte array
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	// Convert the bytes into a struct
	json.Unmarshal(byteValue, &creds)

	// Close the file
	jsonFile.Close()

	// Binding the end-points to their functions
	bindEndPointsToRoutes(creds.Username, creds.Password)

	// Starting to listen
	log.Println("Starting server at port 6969")

	// Try to listen on the port and serve clients
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Println("Something went wrong while listening and serving!")
		log.Fatalln(err)
	}

}
