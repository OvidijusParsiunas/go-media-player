package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/olivere/elastic"
)

type VideoRepository interface {
	Upload(*File, context.Context)
	Download()
}

type LocalVideoRepository struct {
	fileSystem    *FileSystem
	elasticSearch *Elasticsearch
}

type FileSystem struct {
	loc string
}

type Elasticsearch struct {
	client *elastic.Client
}

func NewLocalVideoRepository(fileSystem *FileSystem, elasticSearch *Elasticsearch) *LocalVideoRepository {
	return &LocalVideoRepository{
		fileSystem:    fileSystem,
		elasticSearch: elasticSearch,
	}
}

func NewFileSystem(location string) *FileSystem {
	log.Print("New FileSystem created")
	return &FileSystem{
		loc: location,
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
	localVideoRepo.fileSystem.SaveVideo(file)

	// Do Elasticsearch stuff
	localVideoRepo.elasticSearch.SaveMetaData(file, ctx)
}

func (filesystem *FileSystem) Download() {
	log.Print("FileSystem Download called")
}

func (localVideoRepo *LocalVideoRepository) Download() {
	log.Print("LocalVideoRepository downlaod method called")
}

func (filesystem *FileSystem) SaveVideo(file *File) {
	log.Print(filesystem.loc)

	newPath := fmt.Sprintf("%s/%s.%s", filesystem.loc, file.id, file.ext)
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
