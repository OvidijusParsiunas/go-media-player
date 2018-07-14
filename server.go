package main

import (
	"context"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/olivere/elastic"
	"log"
	"net/http"
	"time"
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
	FileId string
}

// Abstract idea of what a repository for the VideoMeta should be able to do
type VideoMetaRepository interface {
	CreateEntry(context.Context, *VideoMeta) error
	retrieveByFileId(context.Context, string) (*VideoMeta, error)
	search(context.Context, string) ([]VideoMeta, error)
}

// A concrete implementation of the VideoMetaRepository, backed up by elastic search
type ElasticVideoMetaRepository struct {
	client *elastic.Client
}

// The ID of an uploaded file
type FileId struct {
	Id string
}

type FileUploadResponse struct {
	Id        string
	MetaToken string
}

// Called in response to uploading a new file
func UploadFile(videoRepo VideoRepository) WrappedHandler {
	return func(response http.ResponseWriter, request *http.Request) {
		log.Print("Upload file called")

		// Parse the entered values for the form - i think 1024 is the packet size for the video to upload with?
		err := request.ParseMultipartForm(1024)
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
		fileHandle, err := videoRepo.Upload(request.Context(), &file)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"FileID": fileHandle.Id,
			"nbf":    time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString([]byte("abc123"))

		log.Print(tokenString, err)

		r := FileUploadResponse{fileHandle.Id, tokenString}

		response.Header().Set("Content-Type", "application/json")
		json.NewEncoder(response).Encode(r)
	}
}

type UploadMetaBody struct {
	MetaToken string
	Title     string
}

// Called in response to uploading the metadata for an already  uploaded file
func UploadMeta(videoMetaRepo VideoMetaRepository) WrappedHandler {
	return func(response http.ResponseWriter, request *http.Request) {
		log.Print("Upload meta called")

		// get the values
		decoder := json.NewDecoder(request.Body)
		var fields UploadMetaBody
		err := decoder.Decode(&fields)
		if err != nil {
			panic(err)
		}
		log.Println(fields.Title)
		log.Println(fields.MetaToken)

		token, err := jwt.Parse(fields.MetaToken, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, nil
			}

			return []byte("abc123"), nil
		})

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			log.Print(claims["FileID"], claims["nbf"])
			meta := VideoMeta{fields.Title, claims["FileID"].(string)}
			log.Print("meta: ", meta)
			// store it into the meta repo
			videoMetaRepo.CreateEntry(request.Context(), &meta)

			response.Header().Set("Content-Type", "application/json")
			// json.NewEncoder(response).Encode(uploadId)
		} else {
			log.Print(err)
		}

	}
}

// Get back meta info for a video ID
func RetrieveMeta(videoMetaRepo VideoMetaRepository) WrappedHandler {
	return func(response http.ResponseWriter, request *http.Request) {
		log.Print("Getting meta")

		// Get URL variables defined in the router
		vars := mux.Vars(request)

		meta, err := videoMetaRepo.retrieveByFileId(request.Context(), vars["id"])
		if err != nil {
			http.Error(response, "Internal error", 500)
		} else if meta == nil {
			http.NotFound(response, request)
		} else {
			log.Printf("Getting meta for ID: `%s`", vars["id"])
			response.Header().Set("Content-Type", "application/json")
			json.NewEncoder(response).Encode(meta)
		}

	}
}

// When searching for video metas by title
func Search(videoMetaRepo VideoMetaRepository) WrappedHandler {
	return func(response http.ResponseWriter, request *http.Request) {
		log.Print("Search called")

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
		if !createIndex.Acknowledged {
		} // not sure what this means...
	}

	return &ElasticVideoMetaRepository{client: client}
}

// Create the meta entry - haven't checked if this is working yet...
func (self *ElasticVideoMetaRepository) CreateEntry(ctx context.Context, videoMeta *VideoMeta) error {
	put1, err := self.client.Index().
		Index("video_meta").
		Type("video").
		Id(videoMeta.FileId).
		BodyJson(videoMeta).
		Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	log.Printf("Indexed video %s to index %s, type %s\n", put1.Id, put1.Index, put1.Type)
	return nil
}

// Used to get the video meta including title. TODO  && err.(*elastic.Error).Status != 404 could probably be refactored a bit.
func (self *ElasticVideoMetaRepository) retrieveByFileId(ctx context.Context, fileId string) (*VideoMeta, error) {
	log.Printf("retrieving video meta")
	get1, err := self.client.Get().
		Index("video_meta").
		Type("video").
		Id(fileId).
		Do(ctx)
	if err != nil {
		if err.(*elastic.Error).Status != 404 {
			log.Printf("error load meta", err)
			return nil, err
		} else {
			return nil, nil
		}
	} else if get1.Found {
		log.Printf("Got document %s in version %d from index %s, type %s\n", get1.Id, get1.Version, get1.Index, get1.Type)
		var meta VideoMeta
		err = json.Unmarshal(*get1.Source, &meta)
		if err != nil {
			return nil, err
		}
		return &meta, nil
	} else {
		return nil, nil
	}
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
	log.Print("Starting server...")

	// Used while setting up Elastic Client
	ctx := context.Background()

	//connect to elasticsearch on localhost:9200
	elasticClient, err := elastic.NewClient()
	if err != nil {
		panic(err)
	}

	// Set up an elasticsearch video meta repo
	videoMetaRepo := NewElasticVideoMetaRepository(ctx, elasticClient)

	// Set up a local video repo in then "videos" directory
	videoRepo := NewLocalVideoRepository("videos")

	// Using a router lets us be more flexible with URL variables
	router := mux.NewRouter()
	router.HandleFunc("/video/{id}", videoServer(videoRepo))
	router.HandleFunc("/upload/file", UploadFile(videoRepo))
	router.HandleFunc("/upload/meta", UploadMeta(videoMetaRepo))
	router.HandleFunc("/meta/{id}", RetrieveMeta(videoMetaRepo))
	router.HandleFunc("/search", Search(videoMetaRepo))

	// Serve static files to the client
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))

	// The router handles all requests, then passes them along to the appropriate function
	http.Handle("/", router)

	// Listens on port 8080
	log.Fatal(http.ListenAndServe(":8080", nil))
}
