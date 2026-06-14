package fileserver

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/markdown"
	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/internal/testutil"
	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	apiv1service "github.com/usememos/memos/server/router/api/v1"
	"github.com/usememos/memos/store"
	teststore "github.com/usememos/memos/store/test"
)

func TestServeAttachmentFile_ShareTokenAllowsDirectMemoAttachment(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "share-parent-owner",
		Role:     store.RoleUser,
		Email:    "share-parent-owner@example.com",
	})
	require.NoError(t, err)

	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "memo.txt",
			Type:     "text/plain",
			Content:  []byte("memo attachment"),
		},
	})
	require.NoError(t, err)

	parentMemo, err := svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "shared parent",
			Visibility: apiv1.Visibility_PROTECTED,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	share, err := svc.CreateMemoShare(creatorCtx, &apiv1.CreateMemoShareRequest{
		Parent:    parentMemo.Name,
		MemoShare: &apiv1.MemoShare{},
	})
	require.NoError(t, err)
	shareToken := share.Name[strings.LastIndex(share.Name, "/")+1:]

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?share_token=%s", attachment.Name, attachment.Filename, shareToken), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "memo attachment", rec.Body.String())
}

func TestServeAttachmentFile_LocalStaticFileSupportsRangeRequests(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "range-owner",
		Role:     store.RoleUser,
		Email:    "range-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "range.txt",
			Type:     "text/plain",
			Content:  []byte("0123456789"),
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "range memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s", attachment.Name, attachment.Filename), nil)
	req.Header.Set("Range", "bytes=2-5")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusPartialContent, rec.Code)
	require.Equal(t, "2345", rec.Body.String())
	require.Equal(t, "bytes 2-5/10", rec.Header().Get("Content-Range"))
}

func TestServeAttachmentFile_ShareTokenRejectsCommentAttachment(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "private-parent-owner",
		Role:     store.RoleUser,
		Email:    "private-parent-owner@example.com",
	})
	require.NoError(t, err)

	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)
	commenter, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "share-commenter",
		Role:     store.RoleUser,
		Email:    "share-commenter@example.com",
	})
	require.NoError(t, err)
	commenterCtx := context.WithValue(ctx, auth.UserIDContextKey, commenter.ID)

	parentMemo, err := svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "shared parent",
			Visibility: apiv1.Visibility_PROTECTED,
		},
	})
	require.NoError(t, err)

	commentAttachment, err := svc.CreateAttachment(commenterCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "comment.txt",
			Type:     "text/plain",
			Content:  []byte("comment attachment"),
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemoComment(commenterCtx, &apiv1.CreateMemoCommentRequest{
		Name: parentMemo.Name,
		Comment: &apiv1.Memo{
			Content:    "comment with attachment",
			Visibility: apiv1.Visibility_PROTECTED,
			Attachments: []*apiv1.Attachment{
				{Name: commentAttachment.Name},
			},
		},
	})
	require.NoError(t, err)

	share, err := svc.CreateMemoShare(creatorCtx, &apiv1.CreateMemoShareRequest{
		Parent:    parentMemo.Name,
		MemoShare: &apiv1.MemoShare{},
	})
	require.NoError(t, err)
	shareToken := share.Name[strings.LastIndex(share.Name, "/")+1:]

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?share_token=%s", commentAttachment.Name, commentAttachment.Filename, shareToken), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestServeAttachmentFile_MotionClip(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "motion-owner",
		Role:     store.RoleUser,
		Email:    "motion-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "motion.jpg",
			Type:     "image/jpeg",
			Content:  testutil.BuildMotionPhotoJPEG(),
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "motion memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?motion=true", attachment.Name, attachment.Filename), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "video/mp4", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "ftyp")
}

func TestServeAttachmentFile_MotionClipUsesPersistentCache(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "motion-cache-owner",
		Role:     store.RoleUser,
		Email:    "motion-cache-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "motion-cache.jpg",
			Type:     "image/jpeg",
			Content:  testutil.BuildMotionPhotoJPEG(),
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "motion cache memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?motion=true", attachment.Name, attachment.Filename), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	motionPath := filepath.Join(fs.Profile.Data, MotionCacheFolder, attachmentUIDFromName(attachment.Name)+".mp4")
	require.FileExists(t, motionPath)
	require.Contains(t, string(mustReadFile(t, motionPath)), "ftyp")

	cachedBody := []byte("cached-motion")
	require.NoError(t, os.WriteFile(motionPath, cachedBody, 0644))

	secondReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?motion=true", attachment.Name, attachment.Filename), nil)
	secondRec := httptest.NewRecorder()
	e.ServeHTTP(secondRec, secondReq)
	require.Equal(t, http.StatusOK, secondRec.Code)
	require.Equal(t, cachedBody, secondRec.Body.Bytes())
}

func TestServeAttachmentFile_AndroidMotionPhotoThumbnailGeneratesPreviewAndCaches(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "motion-thumb-owner",
		Role:     store.RoleUser,
		Email:    "motion-thumb-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	imageContent := buildDecodableMotionPhotoJPEG(t)
	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "motion-thumb.jpg",
			Type:     "image/jpeg",
			Content:  imageContent,
		},
	})
	require.NoError(t, err)
	require.Equal(t, apiv1.MotionMediaFamily_ANDROID_MOTION_PHOTO, attachment.MotionMedia.GetFamily())

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "motion thumbnail memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?thumbnail=true", attachment.Name, attachment.Filename), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	require.NotEqual(t, imageContent, rec.Body.Bytes())
	_, format, err := image.Decode(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)

	thumbnailPath := filepath.Join(fs.Profile.Data, ThumbnailCacheFolder, attachmentUIDFromName(attachment.Name)+".v3.jpeg")
	require.FileExists(t, thumbnailPath)
	require.Equal(t, rec.Body.Bytes(), mustReadFile(t, thumbnailPath))
}

func TestServeAttachmentFile_SVGThumbnailServedAsImageWithSecurityHeaders(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "svg-owner",
		Role:     store.RoleUser,
		Email:    "svg-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40"><text x="0" y="20">memos</text></svg>`)
	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "preview.svg",
			Type:     "image/svg+xml",
			Content:  svgContent,
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "svg memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?thumbnail=true", attachment.Name, attachment.Filename), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/svg+xml", rec.Header().Get("Content-Type"))
	require.Empty(t, rec.Header().Get("Content-Disposition"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "default-src 'none'; style-src 'unsafe-inline';", rec.Header().Get("Content-Security-Policy"))
	require.Equal(t, svgContent, rec.Body.Bytes())
}

func TestServeAttachmentFile_ThumbnailWithSensitiveMetadataGeneratesPreview(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()

	creator, err := svc.Store.CreateUser(ctx, &store.User{
		Username: "hdr-owner",
		Role:     store.RoleUser,
		Email:    "hdr-owner@example.com",
	})
	require.NoError(t, err)
	creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)

	imageContent := testPNGWithChunk(t, "cICP", []byte{9, 16, 9, 1})
	attachment, err := svc.CreateAttachment(creatorCtx, &apiv1.CreateAttachmentRequest{
		Attachment: &apiv1.Attachment{
			Filename: "hdr.png",
			Type:     "image/png",
			Content:  imageContent,
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateMemo(creatorCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{
			Content:    "hdr memo",
			Visibility: apiv1.Visibility_PUBLIC,
			Attachments: []*apiv1.Attachment{
				{Name: attachment.Name},
			},
		},
	})
	require.NoError(t, err)

	e := echo.New()
	fs.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file/%s/%s?thumbnail=true", attachment.Name, attachment.Filename), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	require.NotEqual(t, imageContent, rec.Body.Bytes())
	decoded, format, err := image.Decode(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, 1, decoded.Bounds().Dx())
	require.Equal(t, 1, decoded.Bounds().Dy())
}

func testPNGWithChunk(t *testing.T, chunkType string, chunkData []byte) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, img))

	pngData := encoded.Bytes()
	iendIndex := bytes.LastIndex(pngData, []byte("IEND"))
	require.GreaterOrEqual(t, iendIndex, 4)

	chunkStart := iendIndex - 4
	var chunk bytes.Buffer
	require.NoError(t, binary.Write(&chunk, binary.BigEndian, uint32(len(chunkData))))
	chunk.WriteString(chunkType)
	chunk.Write(chunkData)
	checksum := crc32.ChecksumIEEE(append([]byte(chunkType), chunkData...))
	require.NoError(t, binary.Write(&chunk, binary.BigEndian, checksum))

	result := make([]byte, 0, len(pngData)+chunk.Len())
	result = append(result, pngData[:chunkStart]...)
	result = append(result, chunk.Bytes()...)
	result = append(result, pngData[chunkStart:]...)
	return result
}

func buildDecodableMotionPhotoJPEG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{R: uint8(40 * x), G: uint8(40 * y), B: 128, A: 255})
		}
	}

	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}))

	jpegData := encoded.Bytes()
	require.True(t, bytes.HasPrefix(jpegData, []byte{0xFF, 0xD8}))

	xmp := []byte(`<?xpacket begin=""?><rdf:Description GCamera:MotionPhoto="1" GCamera:MotionPhotoPresentationTimestampUs="123456"></rdf:Description>`)
	var app1 bytes.Buffer
	app1.Write([]byte{0xFF, 0xE1})
	require.NoError(t, binary.Write(&app1, binary.BigEndian, uint16(len(xmp)+2)))
	app1.Write(xmp)

	result := make([]byte, 0, len(jpegData)+app1.Len()+16)
	result = append(result, jpegData[:2]...)
	result = append(result, app1.Bytes()...)
	result = append(result, jpegData[2:]...)
	result = append(result, []byte{0x00, 0x00, 0x00, 0x10, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0x00, 0x00, 0x00, 0x00}...)
	return result
}

func attachmentUIDFromName(name string) string {
	return name[strings.LastIndex(name, "/")+1:]
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	blob, err := os.ReadFile(path)
	require.NoError(t, err)
	return blob
}

func newShareAttachmentTestServices(ctx context.Context, t *testing.T) (*apiv1service.APIV1Service, *FileServerService, *store.Store, func()) {
	t.Helper()

	testStore := teststore.NewTestingStore(ctx, t)
	testProfile := &profile.Profile{
		Demo:        true,
		Version:     "test-1.0.0",
		InstanceURL: "http://localhost:8080",
		Driver:      "sqlite",
		DSN:         ":memory:",
		Data:        t.TempDir(),
	}
	secret := "test-secret"
	markdownService := markdown.NewService(markdown.WithTagExtension())
	apiService := &apiv1service.APIV1Service{
		Secret:          secret,
		Profile:         testProfile,
		Store:           testStore,
		MarkdownService: markdownService,
		SSEHub:          apiv1service.NewSSEHub(),
	}
	fileService := NewFileServerService(testProfile, testStore, secret)

	return apiService, fileService, testStore, func() {
		testStore.Close()
	}
}
