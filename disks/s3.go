package disks

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3 struct {
	uploader *s3manager.Uploader
	client   *s3.S3
	baseUrl  string
	bucket   string
}

func NewS3(endpoint, region, baseUrl, bucket, keyId, keySecret string) (*S3, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region:      &region,
		Endpoint:    &endpoint,
		Credentials: credentials.NewStaticCredentials(keyId, keySecret, ""),
	})
	if err != nil {
		return nil, err
	}
	disk := S3{
		uploader: s3manager.NewUploader(awsSession),
		client:   s3.New(awsSession),
		baseUrl:  baseUrl,
		bucket:   bucket,
	}
	return &disk, nil
}

func (s *S3) Put(location string, content io.Reader) error {
	acl := "public-read"
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: &s.bucket,
		Key:    &location,
		Body:   content,
		ACL:    &acl,
	})
	if err != nil {
		return fmt.Errorf("Failed to put to location '%s': %w", location, err)
	}
	return nil
}

func (s *S3) Get(location string) (content io.Reader, err error) {
	panic("not implemented") // TODO: Implement
}

func (s *S3) Url(location string) (url string, err error) {
	return fmt.Sprintf("%s/%s", s.baseUrl, location), nil
}
