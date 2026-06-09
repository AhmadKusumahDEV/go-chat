package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioStorage struct {
	client  *minio.Client
	baseURL string
}

func NewMinioStorage(cfg config.Cfg) (ObjectStorage, error) {
	if cfg.Minio.MaxRetries == 0 {
		cfg.Minio.MaxRetries = 3
	}
	if cfg.Minio.ConnectTimeout == 0 {
		cfg.Minio.ConnectTimeout = 10 * time.Second
	}

	customTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.Minio.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	options := &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.Minio.AccessKeyID, cfg.Minio.SecretAccessKey, ""),
		Secure:       cfg.Minio.UseSSL,
		Region:       cfg.Minio.Region,
		Transport:    customTransport,
		BucketLookup: minio.BucketLookupAuto,
	}

	client, err := minio.New(cfg.Minio.Endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio client: %w", err)
	}

	client.SetAppInfo("backend services chat app", "1.0.1")

	return &minioStorage{client: client, baseURL: cfg.Minio.BaseUrl}, nil
}

func (m *minioStorage) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	bucket, err := m.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("minio connection health check failed: %w", err)
	}

	log.Println(bucket)
	return nil
}

func (m *minioStorage) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to verify bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to auto-create bucket: %w", err)
		}
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err = m.client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
	if err != nil {
		return fmt.Errorf("failed to upload object [%s]: %w", objectName, err)
	}

	return nil
}

func (m *minioStorage) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	err := m.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object: %w", err)
	}
	return nil
}

func (m *minioStorage) GetObjectBySignedURL(ctx context.Context, bucketName string, objectName string, expired time.Duration) (string, error) {
	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, objectName, expired, nil)
	if err != nil {
		return "", errors.New("failed generate presigned")
	}

	if m.baseURL != "" {
		parsedBase, err := url.Parse(m.baseURL)
		if err != nil {
			return "", err
		}
		presignedURL.Host = parsedBase.Host
		presignedURL.Scheme = parsedBase.Scheme
		presignedURL.Path = path.Join(parsedBase.Path, presignedURL.Path)
	} else {
		return "", errors.New("base url is empty")
	}

	return presignedURL.String(), nil
}
