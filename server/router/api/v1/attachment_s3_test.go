package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

func TestBuildAttachmentExternalLinkPrefersPublicURLBase(t *testing.T) {
	attachment := &store.Attachment{
		StorageType: storepb.AttachmentStorageType_S3,
		Reference:   "https://bucket.oss-cn-hangzhou.aliyuncs.com/assets/photo.jpg?X-Amz-Signature=old",
		Payload: &storepb.AttachmentPayload{
			Payload: &storepb.AttachmentPayload_S3Object_{
				S3Object: &storepb.AttachmentPayload_S3Object{
					Key: "assets/photo.jpg",
				},
			},
		},
	}
	fallback := &storepb.StorageS3Config{
		PublicUrlBase: "https://media.example.com/",
	}

	require.Equal(t, "https://media.example.com/assets/photo.jpg", buildAttachmentExternalLink(attachment, fallback))
}

func TestBuildAttachmentExternalLinkFallsBackToReference(t *testing.T) {
	attachment := &store.Attachment{
		StorageType: storepb.AttachmentStorageType_S3,
		Reference:   "https://bucket.example.com/assets/photo.jpg?signature=old",
		Payload: &storepb.AttachmentPayload{
			Payload: &storepb.AttachmentPayload_S3Object_{
				S3Object: &storepb.AttachmentPayload_S3Object{
					Key: "assets/photo.jpg",
				},
			},
		},
	}

	require.Equal(t, attachment.Reference, buildAttachmentExternalLink(attachment, nil))
}
