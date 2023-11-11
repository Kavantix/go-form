package disks

import (
	"fmt"

	"github.com/Kavantix/go-form/interfaces"
)

func NewDOSpaces(region, bucket, keyId, keySecret string) (interfaces.Disk, error) {
	return NewS3(
		fmt.Sprintf("https://%s.digitaloceanspaces.com", region),
		region,
		fmt.Sprintf("https://%s.%s.digitaloceanspaces.com", region, bucket),
		bucket,
		keyId,
		keySecret,
	)
}
