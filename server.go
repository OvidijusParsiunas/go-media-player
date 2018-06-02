package main

import (
    "log"
    "net/http"
    mux "github.com/gorilla/mux"
)

//dummy function that gets a video filename by ID. In reality we'd do a database call here
func getVideoById(id string) string {
	if id == "hello" {
		return "video.mp4"
	} else {
		return ""
	}
}

//serve video file to client
func videoServer(response http.ResponseWriter, request *http.Request) {
	//get URL variables defined in the router
	vars := mux.Vars(request)

	//get the filename associated with our ID according to our 'database'
	filename := getVideoById(vars["id"])

	//serve the file and log
	http.ServeFile(response, request, filename)
	log.Printf("Serving video with ID: `%s`, Filename: `%s`", vars["id"], filename)
}

//initiate the http server with a '/' endpoint which will call the serveIndex function
func main() {
	// Using a router lets us be more flexible with URL variables
	router := mux.NewRouter()
  router.HandleFunc("/video/{id}", videoServer)

  //Serve static files to the client
  router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	// the router handles all requests, then passes them along to the appropriate function
	http.Handle("/", router)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
