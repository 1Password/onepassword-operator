package model

import (
	"errors"
)

// File represents a file stored in 1Password.
type File struct {
	ID          string
	Name        string
	Size        int
	ContentPath string
	content     []byte
}

// Content returns the content of the file if they have been loaded and returns an error if they have not been loaded.
// Use `client.GetFileContent(file *File)` instead to make sure the content is fetched automatically if not present.
func (f *File) Content() ([]byte, error) {
	if f.content == nil {
		return nil, errors.New("file content not loaded")
	}
	return f.content, nil
}

func (f *File) SetContent(content []byte) {
	f.content = content
}
