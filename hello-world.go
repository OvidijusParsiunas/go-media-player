package main

import (
    "log"
    "net/http"
)

func serveIndex(w http.ResponseWriter, r *http.Request){
  http.ServeFile(w, r, "./index.html")
  log.Print("/ called and index.html served")
}

func main() {
    http.HandleFunc("/", serveIndex)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
