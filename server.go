package main

import (
    "bytes"
    "context"
    "io"
    "log"
    "net/http"
    "strings"
    "time"
    "github.com/gorilla/mux"
    "github.com/satori/go.uuid"
    "github.com/olivere/elastic"
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
func videoServer(videoRepo VideoRepository) WrappedHandler {
    return func(response http.ResponseWriter, request *http.Request) {
        //get URL variables defined in the router
        vars := mux.Vars(request)

        content := videoRepo.GetContent(vars["id"])
        
        http.ServeContent(response, request, "", time.Time{}, content)

        log.Printf("Serving video with ID: `%s`", vars["id"])
    }
    
}


// A type which contains metadata to do with the video
type VideoMeta struct {
    Title  string
    FileID string
}


// Abstract idea of what a repository for the VideoMeta should do
type VideoMetaRepository interface{
    store(context.Context, *VideoMeta) error
    retrieveByFileId(context.Context, string) (*VideoMeta, error)
    search(context.Context, string) ([]VideoMeta, error)
}

// A concrete implementation of the VideoMetaRepository, backed up by elastic search
type ElasticVideoMetaRepository struct {
    client *elastic.Client
}


// This File struct will contain all the information needed to store the file in a database
type File struct {
    id    string
    name  string
    ext   string
    bytes []byte
}

// Contructor for the File struct
func NewFile(id, name, ext string, bytes []byte) *File {
    return &File{
        id:    id,
        name:  name,
        ext:   ext,
        bytes: bytes,
    }
}

func UploadRequest(videoMetaRepo VideoMetaRepository, videoRepo VideoRepository) WrappedHandler {
    return func(response http.ResponseWriter, request *http.Request) {
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

        uuid := uuid.NewV4()
        id := uuid.String()
        filename := request.FormValue("title")
        newFile := NewFile(id, filename, ext, buffer.Bytes())

        videoRepo.Upload(newFile, request.Context())
    }
}

type WrappedHandler func(response http.ResponseWriter, request *http.Request)

// func retrieveVideo(videoRepo VideoRepository) WrappedHandler {
//     return func(response http.ResponseWriter, request *http.Request) {

//     }
// }

//initiate the http server with a '/' endpoint which will call the serveIndex function
// func main() {
//     // videoRepo := DummyVideoRepo {}
//     fileSystem := NewFileSystem(".")
//     protocol, host, port := "http", "localhost", 9200
//     elasticSearch := NewElasticsearch(protocol, host, port)
//     videoRepo := NewLocalVideoRepository(fileSystem, elasticSearch)
//     // Using a router lets us be more flexible with URL variables
//     router := mux.NewRouter()
//     router.HandleFunc("/", serveIndex)
//     router.HandleFunc("/video/{id}", videoServer)
//     router.HandleFunc("/upload", UploadRequest(videoRepo))
//     router.HandleFunc("/whatever", retrieveVideo(videoRepo))
//     // the router handles all requests, then passes them along to the appropriate function
//     http.Handle("/", router)
//     log.Fatal(http.ListenAndServe(":8080", nil))

func NewElasticVideoMetaRepository(ctx context.Context, client *elastic.Client) *ElasticVideoMetaRepository {
    //if the index doesn't exist, create it
    exists, err := client.IndexExists("video_meta").Do(ctx)
    if err != nil {
        panic(err)
    }
    if !exists {
        createIndex, err := client.CreateIndex("video_meta").BodyString(mapping).Do(ctx)
        if err != nil {
            panic(err)
        }
        if !createIndex.Acknowledged {} // not sure what this means...
    }
    return &ElasticVideoMetaRepository {client: client}
}

func (self *ElasticVideoMetaRepository) store(ctx context.Context, videoMeta *VideoMeta) error {
    log.Printf("hello")
    put1, err := self.client.Index().
        Index("video_meta").
        Type("video").
        BodyJson(videoMeta).
        Do(ctx)
    if err != nil {
        // Handle error
        panic(err)
    }
    log.Printf("Indexed video %s to index %s, type %s\n", put1.Id, put1.Index, put1.Type)
    return nil
}

func (self *ElasticVideoMetaRepository) retrieveByFileId(ctx context.Context, fileId string) (*VideoMeta, error) {
    log.Printf("retrieving video")
    
    
    return nil, nil
}

func (self *ElasticVideoMetaRepository) search(ctx context.Context, searchQuery string) ([]VideoMeta, error) {
    log.Printf("searching videos")
    
    
    return nil, nil
}

// this is the 'schema' of the object we're storing into Elastic search. It's not required, but will improve performance.
const mapping = `
{
    "settings":{
        "number_of_shards": 1,
        "number_of_replicas": 0
    },
    "mappings":{
        "video":{
            "properties":{
                "title":{
                    "type":"keyword"
                },
                "fileId":{
                    "type":"keyword"
                }
            }
        }
    }
}`

//initiate the http server with a '/' endpoint which will call the serveIndex function
func main() {
    ctx := context.Background()

    //connect to elasticsearch on localhost:9200
    elasticClient, err := elastic.NewClient()
    if err != nil {
        panic(err)
    }

    // set up an elasticsearch video meta repo with a single entry
    videoMetaRepo := NewElasticVideoMetaRepository(ctx, elasticClient)
    videoMeta := VideoMeta {Title:"Me at the zoo", FileID: "abc123"}
    videoMetaRepo.store(ctx, &videoMeta)

    // set up a local video repo with a single entry
    videoRepo := NewLocalVideoRepository("videos")

    // Using a router lets us be more flexible with URL variables
    router := mux.NewRouter()
    router.HandleFunc("/video/{id}", videoServer(videoRepo))
    router.HandleFunc("/upload", UploadRequest(videoMetaRepo, videoRepo))


    //Serve static files to the client
    router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

    // the router handles all requests, then passes them along to the appropriate function
    http.Handle("/", router)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
