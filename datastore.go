package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/olivere/elastic"
)

type VideoRepository interface {
	Upload(*File, context.Context)
	GetContent(string) io.ReadSeeker
}

type LocalVideoRepository struct {
	location string
}

type Elasticsearch struct {
	client *elastic.Client
}

func NewLocalVideoRepository(location string) *LocalVideoRepository {
	return &LocalVideoRepository{
		location: location,
	}
}

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

func (localVideoRepo *LocalVideoRepository) Upload(file *File, ctx context.Context) {
	log.Print("LocalVideoRepository upload method called")
	localVideoRepo.SaveVideo(file)
}

func (filesystem *LocalVideoRepository) GetContent(fileId string) io.ReadSeeker {
	log.Print("FileSystem Download called")
	return nil
}

func (localVideoRepo *LocalVideoRepository) Download() {
	log.Print("LocalVideoRepository downlaod method called")
}

func (localVideoRepo *LocalVideoRepository) SaveVideo(file *File) {
	log.Print(localVideoRepo.location)

	newPath := fmt.Sprintf("%s/%s.%s", localVideoRepo.location, file.id, file.ext)
	log.Printf("New path: %s", newPath)

	newFile, err := os.Create(newPath)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open new file %s", newPath))
	}
	defer newFile.Close()

	bytes, err := newFile.Write(file.bytes)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to copy bytes %s", bytes))
	}
	log.Printf("Copied %d bytes.", bytes)
}

func (elasticSearch *Elasticsearch) SaveMetaData(file *File, ctx context.Context) {
	log.Print("SaveMetaData called")
	client := elasticSearch.client
	index := "File"
	exists, err := client.IndexExists(index).Do(ctx)
	if err != nil {
		log.Print("Error checking for index")
		return
	}
	if exists != true {
		log.Printf("Created new index called %s", index)
		elasticSearch.client.CreateIndex(index).Do(ctx)
	}

	//client.Index().Index("File").BodyJson(file)
}
