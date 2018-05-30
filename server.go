package main

import (
    "log"
    "net/http"
)

//serve index.html file to client
func serveIndex(w http.ResponseWriter, r *http.Request){
  http.ServeFile(w, r, "./index.html")
  log.Print("/ called and index.html served")
}

//serve video file to client
func File(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./video.mp4")
}

//initiate the http server with a '/' endpoint which will call the serveIndex function
func main() {
    http.HandleFunc("/", serveIndex)
    http.HandleFunc("/video", File)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
