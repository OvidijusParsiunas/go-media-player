package main

import (
	"fmt"
	"log"
	"os"
)

type VideoRepository interface {
	Upload(*File)
	Download()
}

type FileSystem struct {
	loc string
}

type DummyVideoRepo struct{}

func (videoRepo DummyVideoRepo) Upload(file *File) {
	log.Print("Dummy Video Repo Upload called")
}

func (videoRepo DummyVideoRepo) Download() {

}

func NewFileSystem(location string) *FileSystem {
	log.Print("New FileSystem created")
	return &FileSystem{
		loc: location,
	}
}

func (filesystem *FileSystem) Upload(file *File) {
	log.Print(filesystem.loc)

	newPath := fmt.Sprintf("%s/%s.%s", filesystem.loc, file.name, file.ext)
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

func (filesystem *FileSystem) Download() {
	log.Print("FileSystem Download called")
}
