package main

import (
    "log"
    "net/http"
	mux "github.com/gorilla/mux"
	"bytes"
	"io"
	"strings"
	"github.com/satori/go.uuid"
)

//dummy function that gets a video filename by ID. In reality we'd do a database call here
func getVideoById(id string) string {
	if id == "hello" {
		return "video.mp4"
	} else {
		return ""
	}
}

//serve index.html file to client
func serveIndex(response http.ResponseWriter, request *http.Request){
  http.ServeFile(response, request, "./index.html")
  log.Print("/ called and index.html served")
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

type File struct {
	name string
	ext string
	bytes []byte
}

func NewFile(name, ext string, bytes []byte) *File {
	return &File {
		name: name,
		ext: ext,
		bytes: bytes,
	}
}

func UploadRequest(videoRepo VideoRepository) WrappedHandler {
	return func (response http.ResponseWriter, request *http.Request) {
		log.Print("Upload called")
		file, headers, err := request.FormFile("upload")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		buffer := bytes.NewBuffer(nil)
		numOfBytes, err := io.Copy(buffer, file)
		if err != nil {
			panic(err)
		}
		log.Printf("%d bytes copied", numOfBytes)
		
		ext := strings.Split(headers.Filename, ".")[1]
		
		id, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}

		filename := id.String()
		newFile := NewFile(filename, ext, buffer.Bytes())

		videoRepo.Upload(newFile)
	}
}

type WrappedHandler func (response http.ResponseWriter, request *http.Request)

func retrieveVideo(videoRepo VideoRepository) WrappedHandler {
	return func (response http.ResponseWriter, request *http.Request) {
		
	}
}

//initiate the http server with a '/' endpoint which will call the serveIndex function
func main() {
	// videoRepo := DummyVideoRepo {}
	videoRepo := NewFileSystem(".")
	// Using a router lets us be more flexible with URL variables
	router := mux.NewRouter()
	router.HandleFunc("/", serveIndex)
	router.HandleFunc("/video/{id}", videoServer)
	router.HandleFunc("/upload", UploadRequest(videoRepo))
	router.HandleFunc("/whatever", retrieveVideo(videoRepo))
	// the router handles all requests, then passes them along to the appropriate function
	http.Handle("/", router)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
