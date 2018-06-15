package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"
    "github.com/gorilla/mux"
    "github.com/satori/go.uuid"
    "github.com/olivere/elastic"
)

// A convenience thing - just an http-handling function
type WrappedHandler func(response http.ResponseWriter, request *http.Request)


//serve video file to client
func videoServer(videoRepo VideoRepository) WrappedHandler {
    return func(response http.ResponseWriter, request *http.Request) {
        // Get URL variables defined in the router
        vars := mux.Vars(request)

        // Get a file-like object to serve
        content := videoRepo.GetContent(vars["id"])
        
        // Serve the file
        http.ServeContent(response, request, "", time.Time{}, content)

        log.Printf("Serving video with ID: `%s`", vars["id"])
    }
}


// A type which contains metadata to do with the video
type VideoMeta struct {
    Title  string
    FileID string
}


// Abstract idea of what a repository for the VideoMeta should be able to do
type VideoMetaRepository interface{
    CreateEntry(context.Context, string) (*VideoMeta, error)
    retrieveByFileId(context.Context, string) (*VideoMeta, error)
    search(context.Context, string) ([]VideoMeta, error)
}


// A concrete implementation of the VideoMetaRepository, backed up by elastic search
type ElasticVideoMetaRepository struct {
    client *elastic.Client
}


// Called in response to uploading a new file
func UploadRequest(videoMetaRepo VideoMetaRepository, videoRepo VideoRepository) WrappedHandler {
    return func(response http.ResponseWriter, request *http.Request) {
        log.Print("Upload called")

        // Parse the entered values for the form - i think 1024 is the packet size for the video to upload with?
        err := request.ParseMultipartForm(1024)
        if err != nil {
            panic(err)
        }

        // The title of the uploaded video
        title := request.PostFormValue("title")

        // Create the metadata entry, currently we only store the Title. This generates an ID for us.
        videoMeta, err := videoMetaRepo.CreateEntry(request.Context(), title)
        if err != nil {
            panic(err)
        }

        // Get the uploaded file
        file, _, err := request.FormFile("upload")        
        if err != nil {
            panic(err)
        }
        defer file.Close()

        // Store the file in the video repository, with the FileID of the given metadata
        videoRepo.Upload(request.Context(), &file, videoMeta)

        // Currently just redirecting directly to the video - in future to a proper page.
        http.Redirect(response, request, fmt.Sprintf("/video/%s", videoMeta.FileID), http.StatusSeeOther)
    }
}


// Create the elastic video meta repository
func NewElasticVideoMetaRepository(ctx context.Context, client *elastic.Client) *ElasticVideoMetaRepository {

    // Check if the index exists
    exists, err := client.IndexExists("video_meta").Do(ctx)
    if err != nil {
        panic(err)
    }

    // If not, create it
    if !exists {
        createIndex, err := client.CreateIndex("video_meta").BodyString(mapping).Do(ctx)
        if err != nil {
            panic(err)
        }
        if !createIndex.Acknowledged {} // not sure what this means...
    }

    return &ElasticVideoMetaRepository {client: client}
}

// Create the meta entry - haven't checked if this is working yet...
func (self *ElasticVideoMetaRepository) CreateEntry(ctx context.Context, name string) (*VideoMeta, error) {

    id := uuid.NewV4().String()
    videoMeta := VideoMeta {
        Title: name,
        FileID: id,
    }

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
    return &videoMeta, nil
}

// Needs some work still - would be used to get the video meta including title.
func (self *ElasticVideoMetaRepository) retrieveByFileId(ctx context.Context, fileId string) (*VideoMeta, error) {
    log.Printf("retrieving video")
    return nil, nil
}

// Needs implemented - would return a list of videos relevant to the searchQuery
func (self *ElasticVideoMetaRepository) search(ctx context.Context, searchQuery string) ([]VideoMeta, error) {
    log.Printf("searching videos")
    return nil, nil
}

// This is the 'schema' of the object we're storing into Elastic search. It's not required, but will improve performance. Should be moved elsewhere though.
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


// Initiate the http server
func main() {

	// Used while setting up Elastic Client
    ctx := context.Background()

    //connect to elasticsearch on localhost:9200
    elasticClient, err := elastic.NewClient()
    if err != nil {
        panic(err)
    }

    // Set up an elasticsearch video meta repo
    videoMetaRepo := NewElasticVideoMetaRepository(ctx, elasticClient)

    // Set up a local video repo in the "videos" directory
    videoRepo := NewLocalVideoRepository("videos")

    // Using a router lets us be more flexible with URL variables
    router := mux.NewRouter()
    router.HandleFunc("/video/{id}", videoServer(videoRepo))
    router.HandleFunc("/upload", UploadRequest(videoMetaRepo, videoRepo))


    // Serve static files to the client
    router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

    // The router handles all requests, then passes them along to the appropriate function
    http.Handle("/", router)

    // Listens on port 8080
    log.Fatal(http.ListenAndServe(":8080", nil))
}
