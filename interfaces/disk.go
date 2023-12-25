package interfaces

import "io"

type Disk interface {
	Exists(location string) (exists bool, err error)
	Put(location string, content io.Reader) error
	Get(location string) (content io.Reader, err error)
	Url(location string) (url string, err error)
}

type DirectUploadDisk interface {
	Disk
	PutUrl(location string) (url string, err error)
}
