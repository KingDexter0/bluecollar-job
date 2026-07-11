package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	SignedURL(ctx context.Context, documentRef string) (string, error)
}

type LocalObjectStore struct {
	basePath string
}

func NewLocalObjectStore(basePath string) *LocalObjectStore {
	if strings.TrimSpace(basePath) == "" {
		basePath = "./var/uploads"
	}
	return &LocalObjectStore{basePath: basePath}
}

func (s *LocalObjectStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	cleanKey := filepath.Clean(strings.TrimPrefix(key, "/"))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") {
		return "", fmt.Errorf("invalid object key")
	}
	target := filepath.Join(s.basePath, cleanKey)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return "", err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return "", err
	}
	return "local://" + cleanKey, nil
}

func (s *LocalObjectStore) SignedURL(ctx context.Context, documentRef string) (string, error) {
	if !strings.HasPrefix(documentRef, "local://") {
		return "", fmt.Errorf("unsupported local document reference")
	}
	return documentRef, nil
}

type LinodeObjectStore struct {
	bucket string
	client *s3.S3
}

type LinodeObjectStoreConfig struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func NewLinodeObjectStore(cfg LinodeObjectStoreConfig) (*LinodeObjectStore, error) {
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	cfg.SecretAccessKey = strings.TrimSpace(cfg.SecretAccessKey)
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("object storage bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("object storage region is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = fmt.Sprintf("https://%s.linodeobjects.com", cfg.Region)
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("object storage access key and secret are required")
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:           aws.String(cfg.Region),
		Endpoint:         aws.String(cfg.Endpoint),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return &LinodeObjectStore{
		bucket: cfg.Bucket,
		client: s3.New(awsSession),
	}, nil
}

func (s *LinodeObjectStore) Put(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	cleanKey, err := cleanObjectKey(key)
	if err != nil {
		return "", err
	}
	input := &s3.PutObjectInput{
		Bucket:               aws.String(s.bucket),
		Key:                  aws.String(cleanKey),
		Body:                 aws.ReadSeekCloser(reader),
		ContentType:          aws.String(strings.TrimSpace(contentType)),
		ServerSideEncryption: aws.String("AES256"),
	}
	if strings.TrimSpace(contentType) == "" {
		input.ContentType = aws.String("application/octet-stream")
	}
	if _, err := s.client.PutObjectWithContext(ctx, input); err != nil {
		return "", fmt.Errorf("object storage upload failed")
	}
	return "s3://" + s.bucket + "/" + cleanKey, nil
}

func (s *LinodeObjectStore) SignedURL(ctx context.Context, documentRef string) (string, error) {
	prefix := "s3://" + s.bucket + "/"
	if !strings.HasPrefix(documentRef, prefix) {
		return "", fmt.Errorf("unsupported object reference")
	}
	key := strings.TrimPrefix(documentRef, prefix)
	request, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return request.Presign(15 * time.Minute)
}

func cleanObjectKey(key string) (string, error) {
	cleanKey := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(strings.TrimSpace(key), "/")))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "../") || strings.HasPrefix(cleanKey, "/") {
		return "", fmt.Errorf("invalid object key")
	}
	return cleanKey, nil
}
