package disks

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

type localDiskMode bool

const (
	LocalDiskModePublic  = localDiskMode(true)
	LocalDiskModePrivate = localDiskMode(false)
)

type Local struct {
	rootdir     string
	baseUrl     string
	permissions fs.FileMode
}

func NewLocal(rootDirirectory, baseUrl string, mode localDiskMode) *Local {
	disk := Local{
		rootdir:     rootDirirectory,
		permissions: 0700,
	}
	if mode {
		disk.permissions = 0755
	}
	os.MkdirAll(rootDirirectory, disk.permissions)
	return &disk
}

func isValidLocation(location string) error {
	if strings.Contains(location, "..") {
		return errors.New("location contains `..` which might cause access outside of the rootdir")
	}
	if strings.HasPrefix(location, "/") {
		return errors.New("location is an absolute path wich might cause access outside of the rootdir")
	}
	return nil
}

func (l *Local) PathTo(location string) (string, error) {
	err := isValidLocation(location)
	if err != nil {
		return "", err
	}
	builder := strings.Builder{}
	builder.WriteString(l.rootdir)
	builder.WriteRune(os.PathSeparator)
	builder.WriteString(location)
	return builder.String(), nil
}

func (l *Local) Put(location string, content io.Reader) error {
	path, err := l.PathTo(location)
	if err != nil {
		return fmt.Errorf("Failed to put to location '%s': %w", location, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, l.permissions)
	if err != nil {
		return fmt.Errorf("Failed to put to location '%s': %w", location, err)
	}
	defer file.Close()
	_, err = io.Copy(file, content)
	if err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("Failed to put to location '%s': %w", location, err)
	}
	return nil
}

func (l *Local) Get(location string) (io.Reader, error) {
	panic("not implemented") // TODO: Implement
}

func (l *Local) Url(location string) (string, error) {
	err := isValidLocation(location)
	if err != nil {
		return "", fmt.Errorf("Failed to get url for location '%s': %w", location, err)
	}
	builder := strings.Builder{}
	builder.WriteString(l.rootdir)
	builder.WriteByte('/')
	builder.WriteString(location)
	return builder.String(), nil
}
