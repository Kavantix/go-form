package disks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3 struct {
	uploader      *s3manager.Uploader
	client        *s3.Client
	predignClient *s3.PresignClient
	baseUrl       string
	bucket        string
}

func NewS3(endpoint, region, baseUrl, bucket, keyId, keySecret string, usePathStyle bool) (*S3, error) {
	client := s3.New(s3.Options{
		Region:       region,
		BaseEndpoint: &endpoint,
		UsePathStyle: usePathStyle,
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     keyId,
				SecretAccessKey: keySecret,
			}, nil
		}),
	})
	disk := S3{
		uploader:      s3manager.NewUploader(client),
		predignClient: s3.NewPresignClient(client),
		client:        client,
		baseUrl:       baseUrl,
		bucket:        bucket,
	}
	return &disk, nil
}

func (s *S3) basePutObjectInput(location string) *s3.PutObjectInput {

	return &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &location,
		ACL:    s3types.ObjectCannedACLPublicRead,
	}
}

func (s *S3) Put(location string, content io.Reader) error {
	input := s.basePutObjectInput(location)
	input.Body = content
	_, err := s.uploader.Upload(context.TODO(), input)
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

func (s *S3) PutUrl(location string) (string, error) {
	input := s.basePutObjectInput(location)
	req, err := s.predignClient.PresignPutObject(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("Failed to create presigned put to location '%s': %w", location, err)
	}
	return req.URL, nil
}

func (s *S3) Exists(location string) (bool, error) {
	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &location,
	})
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.(type) {
		case *s3types.NotFound:
			return false, nil
		}
	}
	if err != nil {
		return false, fmt.Errorf("Failed to check if location '%s' exists: %w", location, err)
	}
	return true, nil
}
