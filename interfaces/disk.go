package interfaces

import "io"

type Disk interface {
	Put(location string, content io.Reader) error
	Get(location string) (content io.Reader, err error)
	Url(location string) (url string, err error)
}
