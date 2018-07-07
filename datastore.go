package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/olivere/elastic"
	"github.com/satori/go.uuid"
)

// A video repository stores the actualy video files
type VideoRepository interface {
	Upload(context.Context, *multipart.File) (FileId,error)
	GetContent(string) io.ReadSeeker
}

// The local video repository just stores files on your local machine's file system
type LocalVideoRepository struct {
	location string
}

// This seems unnecesary - perhaps we could remove it?
type Elasticsearch struct {
	client *elastic.Client
}


// Fairly basic setup - perhaps could ensure the 'location' exists / is valid?
func NewLocalVideoRepository(location string) *LocalVideoRepository {
	return &LocalVideoRepository{
		location: location,
	}
}

// Connects to elastic server and returns our wrapper thingy - WE'RE NOT EVEN CALLING THIS CURRENTLY
func NewElasticsearch(protocol, host string, port int) *Elasticsearch {
	log.Print("NewElasticSearch called")
	url := fmt.Sprintf("%s://%s:%d", protocol, host, port)
	log.Print(url)
	client, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		log.Fatal("Failed to create elastic client\n\n", err)
	}
	return &Elasticsearch{
		client: client,
	}
}


// 'Uploads' a given file to your file system
func (localVideoRepo *LocalVideoRepository) Upload(ctx context.Context, file *multipart.File) (FileId,error) {
	log.Print("LocalVideoRepository upload method called")
	id := uuid.NewV4().String()
	fileHandle := FileId{id}
	localVideoRepo.SaveVideo(file, fileHandle.Id)
	return fileHandle, nil
}

// Returns a file-like object - in this case an actual file from our filesystem
func (localVideoRepo *LocalVideoRepository) GetContent(fileId string) io.ReadSeeker {
	log.Print("FileSystem Download called")

	file, err := os.Open(fmt.Sprintf("%s/%s", localVideoRepo.location, fileId))
	if err != nil {
		panic(err)
	}
	return file
}

// Saves a video to the local file system
func (localVideoRepo *LocalVideoRepository) SaveVideo(file *multipart.File, saveName string) {
	newPath := fmt.Sprintf("%s/%s", localVideoRepo.location, saveName)
	log.Printf("New path: %s", newPath)

	newFile, err := os.Create(newPath)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open new file %s", newPath))
	}
	defer newFile.Close()

	io.Copy(newFile, *file)
}