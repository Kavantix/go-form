package disks

import (
	"fmt"

	"github.com/Kavantix/go-form/interfaces"
)

func NewDOSpaces(region, bucket, keyId, keySecret string) (interfaces.DirectUploadDisk, error) {
	return NewS3(
		fmt.Sprintf("https://%s.digitaloceanspaces.com", region),
		region,
		fmt.Sprintf("https://%s.%s.digitaloceanspaces.com", bucket, region),
		bucket,
		keyId,
		keySecret,
		true,
	)
}
