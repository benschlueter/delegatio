package main

import (
	"context"
	"io"
	"os"

	"github.com/benschlueter/delegatio/internal/osimage"
	"github.com/benschlueter/delegatio/internal/osimage/gcp"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	zapconf := zap.NewDevelopmentConfig()
	log, err := zapconf.Build()
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create logger")
	}
	defer func() { _ = log.Sync() }()

	uploader, err := gcp.New(ctx, "delegatio", "europe-west3", "delegatio-test-cli", log)
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create uploader")
	}
	file, err := os.Open("/tmp/compressed-image.tar.gz")
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to open image file")
	}
	defer file.Close()
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to get image file size")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to seek to start of image file")
	}
	_, err = uploader.Upload(ctx, &osimage.UploadRequest{
		Size:     size,
		Image:    file,
		Provider: "gcp",
		Version:  "0-0-0",
		Variant:  "test",
	})
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to upload image")
	}
}
