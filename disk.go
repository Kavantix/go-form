package main

import (
	"fmt"
	"log"

	"github.com/Kavantix/go-form/disks"
	"github.com/Kavantix/go-form/interfaces"
)

func ResolveDisk(
	LookupEnv func(key, fallback string) string,
	MustLookupEnv func(key string) string,
) interfaces.Disk {
	var err error
	var disk interfaces.Disk
	uploadDisk := LookupEnv("UPLOAD_DISK", "local")
	switch uploadDisk {
	case "local":
		disk = disks.NewLocal("./storage/public", "/storage", disks.LocalDiskModePublic)
	case "do-spaces":
		disk, err = disks.NewDOSpaces(
			MustLookupEnv("DO_SPACES_REGION"),
			MustLookupEnv("DO_SPACES_BUCKET"),
			MustLookupEnv("DO_SPACES_KEY_ID"),
			MustLookupEnv("DO_SPACES_KEY_SECRET"),
		)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to create do-spaces disk: %w", err))
		}
	case "s3":
		disk, err = disks.NewS3(
			MustLookupEnv("S3_ENDPOINT"),
			MustLookupEnv("S3_REGION"),
			MustLookupEnv("S3_BASE_URL"),
			MustLookupEnv("S3_BUCKET"),
			MustLookupEnv("S3_KEY_ID"),
			MustLookupEnv("S3_KEY_SECRET"),
			false,
		)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to create s3 disk: %w", err))
		}
	default:
		log.Fatalf("UPLOAD_DISK '%s' is not supported, supported: (locale/s3)", uploadDisk)
	}
	return disk
}
