package storage

import (
	"context"
	"io"
	"time"
)

type ObjectStorage interface {
	Ping(ctx context.Context) error
	UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) error
	DeleteObject(ctx context.Context, bucketName, objectName string) error
	GetObjectBySignedURL(ctx context.Context, bucketName string, objectName string, expired time.Duration) (string, error)
	GetObjectURL(ctx context.Context, objectName string, bucketName string) (string, error)
}
