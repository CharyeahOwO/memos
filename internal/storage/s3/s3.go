package s3

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"

	storepb "github.com/usememos/memos/proto/gen/store"
)

// DefaultCacheControl is used for new S3 objects when no cache policy is configured.
const DefaultCacheControl = "public, max-age=31536000, immutable"

type Client struct {
	Client *s3service.Client
	Bucket *string
	config *storepb.StorageS3Config
}

func NewClient(ctx context.Context, s3Config *storepb.StorageS3Config) (*Client, error) {
	s3Config = CloneConfig(s3Config)
	client, err := newAWSClient(ctx, s3Config, EffectiveServerEndpoint(s3Config))
	if err != nil {
		return nil, err
	}
	return &Client{
		Client: client,
		Bucket: aws.String(s3Config.Bucket),
		config: s3Config,
	}, nil
}

func newAWSClient(ctx context.Context, s3Config *storepb.StorageS3Config, endpoint string) (*s3service.Client, error) {
	if s3Config == nil {
		return nil, errors.New("s3 config is required")
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3Config.AccessKeyId, s3Config.AccessKeySecret, "")),
		config.WithRegion(s3Config.Region),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load s3 config")
	}

	client := s3service.NewFromConfig(cfg, func(o *s3service.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = s3Config.UsePathStyle
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	return client, nil
}

// UploadObject uploads an object to S3.
func (c *Client) UploadObject(ctx context.Context, key string, fileType string, content io.Reader) (string, error) {
	putInput := c.newPutObjectInput(key, fileType, content)
	if _, err := c.Client.PutObject(ctx, &putInput); err != nil {
		return "", err
	}
	return key, nil
}

func (c *Client) newPutObjectInput(key string, fileType string, content io.Reader) s3service.PutObjectInput {
	putInput := s3service.PutObjectInput{
		Bucket:      c.Bucket,
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
		Body:        content,
	}
	if cacheControl := EffectiveCacheControl(c.config); cacheControl != "" {
		putInput.CacheControl = aws.String(cacheControl)
	}
	if shouldServeInline(fileType) {
		putInput.ContentDisposition = aws.String("inline")
	}
	return putInput
}

// PresignGetObject presigns an object in S3.
func (c *Client) PresignGetObject(ctx context.Context, key string) (string, error) {
	presignClientTarget := c.Client
	if c.config.GetEndpoint() != "" && c.config.GetEndpoint() != EffectiveServerEndpoint(c.config) {
		client, err := newAWSClient(ctx, c.config, c.config.GetEndpoint())
		if err != nil {
			return "", err
		}
		presignClientTarget = client
	}

	presignClient := s3service.NewPresignClient(presignClientTarget)
	presignResult, err := presignClient.PresignGetObject(ctx, &s3service.GetObjectInput{
		Bucket: aws.String(*c.Bucket),
		Key:    aws.String(key),
	}, func(opts *s3service.PresignOptions) {
		// Set the expiration time of the presigned URL to 5 days.
		// Reference: https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-query-string-auth.html
		opts.Expires = time.Duration(5 * 24 * time.Hour)
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to presign get object")
	}
	return presignResult.URL, nil
}

// GetObject retrieves an object from S3.
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	output, err := c.Client.GetObject(ctx, &s3service.GetObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to download object")
	}
	defer output.Body.Close()
	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read object body")
	}
	return data, nil
}

// GetObjectStream retrieves an object from S3 as a stream.
func (c *Client) GetObjectStream(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := c.Client.GetObject(ctx, &s3service.GetObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get object")
	}
	return output.Body, nil
}

// DeleteObject deletes an object in S3.
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.Client.DeleteObject(ctx, &s3service.DeleteObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete object")
	}
	return nil
}

// CloneConfig returns a writable copy of the S3 config.
func CloneConfig(cfg *storepb.StorageS3Config) *storepb.StorageS3Config {
	if cfg == nil {
		return nil
	}
	return &storepb.StorageS3Config{
		AccessKeyId:      cfg.GetAccessKeyId(),
		AccessKeySecret:  cfg.GetAccessKeySecret(),
		Endpoint:         cfg.GetEndpoint(),
		Region:           cfg.GetRegion(),
		Bucket:           cfg.GetBucket(),
		UsePathStyle:     cfg.GetUsePathStyle(),
		InternalEndpoint: cfg.GetInternalEndpoint(),
		PublicUrlBase:    cfg.GetPublicUrlBase(),
		CacheControl:     cfg.GetCacheControl(),
	}
}

// OverlayConfig fills missing fields in primary from fallback.
func OverlayConfig(primary, fallback *storepb.StorageS3Config) *storepb.StorageS3Config {
	resolved := CloneConfig(primary)
	if resolved == nil {
		return CloneConfig(fallback)
	}
	if fallback == nil {
		return resolved
	}

	if resolved.AccessKeyId == "" {
		resolved.AccessKeyId = fallback.GetAccessKeyId()
	}
	if resolved.AccessKeySecret == "" {
		resolved.AccessKeySecret = fallback.GetAccessKeySecret()
	}
	if resolved.Endpoint == "" {
		resolved.Endpoint = fallback.GetEndpoint()
	}
	if resolved.Region == "" {
		resolved.Region = fallback.GetRegion()
	}
	if resolved.Bucket == "" {
		resolved.Bucket = fallback.GetBucket()
	}
	if resolved.InternalEndpoint == "" {
		resolved.InternalEndpoint = fallback.GetInternalEndpoint()
	}
	if resolved.PublicUrlBase == "" {
		resolved.PublicUrlBase = fallback.GetPublicUrlBase()
	}
	if resolved.CacheControl == "" {
		resolved.CacheControl = fallback.GetCacheControl()
	}
	return resolved
}

// EffectiveServerEndpoint returns the endpoint used by server-side S3 operations.
func EffectiveServerEndpoint(cfg *storepb.StorageS3Config) string {
	if cfg == nil {
		return ""
	}
	if internalEndpoint := strings.TrimSpace(cfg.GetInternalEndpoint()); internalEndpoint != "" {
		return internalEndpoint
	}
	return strings.TrimSpace(cfg.GetEndpoint())
}

// BuildPublicObjectURL returns a stable browser-facing object URL.
func BuildPublicObjectURL(publicURLBase, key string) string {
	publicURLBase = strings.TrimRight(strings.TrimSpace(publicURLBase), "/")
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if publicURLBase == "" || key == "" {
		return ""
	}
	return publicURLBase + "/" + escapeObjectKeyPath(key)
}

// PublicObjectURL returns the stable public object URL for the config, if available.
func PublicObjectURL(cfg *storepb.StorageS3Config, key string) string {
	if cfg == nil {
		return ""
	}
	return BuildPublicObjectURL(cfg.GetPublicUrlBase(), key)
}

// EffectiveCacheControl returns the configured upload cache policy or the media-safe default.
func EffectiveCacheControl(cfg *storepb.StorageS3Config) string {
	if cfg != nil {
		if cacheControl := strings.TrimSpace(cfg.GetCacheControl()); cacheControl != "" {
			return cacheControl
		}
	}
	return DefaultCacheControl
}

func escapeObjectKeyPath(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func shouldServeInline(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/")
}
