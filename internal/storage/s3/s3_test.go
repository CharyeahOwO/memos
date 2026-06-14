package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestBuildPublicObjectURL(t *testing.T) {
	require.Equal(t, "https://media.example.com/assets/photo.jpg", BuildPublicObjectURL("https://media.example.com/", "/assets/photo.jpg"))
	require.Equal(t, "https://media.example.com/assets/My%20Photo.jpg", BuildPublicObjectURL("https://media.example.com", "assets/My Photo.jpg"))
	require.Empty(t, BuildPublicObjectURL("", "assets/photo.jpg"))
	require.Empty(t, BuildPublicObjectURL("https://media.example.com", ""))
}

func TestOverlayConfigFillsBrowserAndServerFields(t *testing.T) {
	primary := &storepb.StorageS3Config{
		Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
		Bucket:   "memos",
	}
	fallback := &storepb.StorageS3Config{
		Endpoint:         "https://fallback.example.com",
		InternalEndpoint: "https://oss-cn-hangzhou-internal.aliyuncs.com",
		PublicUrlBase:    "https://media.example.com",
		CacheControl:     "public, max-age=31536000, immutable",
	}

	resolved := OverlayConfig(primary, fallback)
	require.Equal(t, "https://oss-cn-hangzhou.aliyuncs.com", resolved.Endpoint)
	require.Equal(t, "https://oss-cn-hangzhou-internal.aliyuncs.com", resolved.InternalEndpoint)
	require.Equal(t, "https://media.example.com", resolved.PublicUrlBase)
	require.Equal(t, "public, max-age=31536000, immutable", resolved.CacheControl)
	require.Equal(t, "https://oss-cn-hangzhou-internal.aliyuncs.com", EffectiveServerEndpoint(resolved))
	require.Equal(t, "https://media.example.com/assets/photo.jpg", PublicObjectURL(resolved, "assets/photo.jpg"))
	require.Equal(t, "public, max-age=31536000, immutable", EffectiveCacheControl(resolved))
}

func TestNewClientUsesInternalEndpointForServerOperations(t *testing.T) {
	var requestedPath string
	internalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("ok"))
	}))
	defer internalServer.Close()

	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server-side object read should not use public endpoint: %s", r.URL.String())
	}))
	defer publicServer.Close()

	client, err := NewClient(context.Background(), &storepb.StorageS3Config{
		AccessKeyId:      "AKID",
		AccessKeySecret:  "SECRET",
		Endpoint:         publicServer.URL,
		InternalEndpoint: internalServer.URL,
		Region:           "us-east-1",
		Bucket:           "memos",
		UsePathStyle:     true,
	})
	require.NoError(t, err)

	blob, err := client.GetObject(context.Background(), "assets/photo.jpg")
	require.NoError(t, err)
	require.Equal(t, []byte("ok"), blob)
	require.Equal(t, "/memos/assets/photo.jpg", requestedPath)
}

func TestPresignGetObjectUsesPublicEndpointWhenInternalEndpointConfigured(t *testing.T) {
	client, err := NewClient(context.Background(), &storepb.StorageS3Config{
		AccessKeyId:      "AKID",
		AccessKeySecret:  "SECRET",
		Endpoint:         "https://oss-cn-hangzhou.aliyuncs.com",
		InternalEndpoint: "https://oss-cn-hangzhou-internal.aliyuncs.com",
		Region:           "cn-hangzhou",
		Bucket:           "memos",
		UsePathStyle:     true,
	})
	require.NoError(t, err)

	presignedURL, err := client.PresignGetObject(context.Background(), "assets/photo.jpg")
	require.NoError(t, err)
	require.NotContains(t, presignedURL, "internal")

	parsedURL, err := neturl.Parse(presignedURL)
	require.NoError(t, err)
	require.Equal(t, "oss-cn-hangzhou.aliyuncs.com", parsedURL.Host)
	require.Equal(t, "/memos/assets/photo.jpg", parsedURL.Path)
	require.NotEmpty(t, parsedURL.Query().Get("X-Amz-Signature"))
}

func TestEffectiveCacheControlDefaultsWhenUnset(t *testing.T) {
	require.Equal(t, DefaultCacheControl, EffectiveCacheControl(nil))
	require.Equal(t, DefaultCacheControl, EffectiveCacheControl(&storepb.StorageS3Config{}))
	require.Equal(t, "public, max-age=60", EffectiveCacheControl(&storepb.StorageS3Config{CacheControl: " public, max-age=60 "}))
}

func TestNewPutObjectInputSetsBrowserMediaMetadata(t *testing.T) {
	client := &Client{
		Bucket: aws.String("memos"),
		config: &storepb.StorageS3Config{CacheControl: "public, max-age=60"},
	}

	input := client.newPutObjectInput("assets/photo.jpg", "image/jpeg", strings.NewReader("jpeg"))
	require.Equal(t, "memos", aws.ToString(input.Bucket))
	require.Equal(t, "assets/photo.jpg", aws.ToString(input.Key))
	require.Equal(t, "image/jpeg", aws.ToString(input.ContentType))
	require.Equal(t, "public, max-age=60", aws.ToString(input.CacheControl))
	require.Equal(t, "inline", aws.ToString(input.ContentDisposition))
	require.NotNil(t, input.Body)
}

func TestNewPutObjectInputDoesNotServeDocumentsInline(t *testing.T) {
	client := &Client{
		Bucket: aws.String("memos"),
		config: &storepb.StorageS3Config{},
	}

	input := client.newPutObjectInput("assets/doc.pdf", "application/pdf", strings.NewReader("pdf"))
	require.Equal(t, DefaultCacheControl, aws.ToString(input.CacheControl))
	require.Nil(t, input.ContentDisposition)
}
