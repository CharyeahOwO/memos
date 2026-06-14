import { describe, expect, it } from "vitest";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { MotionMediaFamily, MotionMediaRole } from "@/types/proto/api/v1/attachment_service_pb";
import { buildAttachmentVisualItems } from "@/utils/media-item";

const attachment = (overrides: Partial<Attachment>): Attachment =>
  ({
    name: "attachments/test",
    filename: "test.jpg",
    type: "image/jpeg",
    externalLink: "https://media.example.com/assets/test.jpg",
    ...overrides,
  }) as Attachment;

describe("buildAttachmentVisualItems", () => {
  it("uses thumbnail endpoints for image posters and keeps external links for originals", () => {
    const [item] = buildAttachmentVisualItems([
      attachment({
        name: "attachments/photo",
        filename: "photo.jpg",
        externalLink: "https://media.example.com/assets/photo.jpg",
      }),
    ]);

    expect(item.kind).toBe("image");
    expect(item.sourceUrl).toBe("https://media.example.com/assets/photo.jpg");
    expect(item.posterUrl).toBe(`${window.location.origin}/file/attachments/photo/photo.jpg?thumbnail=true`);
    expect(item.previewItem).toMatchObject({
      kind: "image",
      sourceUrl: "https://media.example.com/assets/photo.jpg",
      posterUrl: `${window.location.origin}/file/attachments/photo/photo.jpg?thumbnail=true`,
    });
  });

  it("uses preview and motion endpoints for Android Motion Photo containers", () => {
    const [item] = buildAttachmentVisualItems([
      attachment({
        name: "attachments/motion",
        filename: "motion.jpg",
        externalLink: "https://media.example.com/assets/motion.jpg",
        motionMedia: {
          family: MotionMediaFamily.ANDROID_MOTION_PHOTO,
          role: MotionMediaRole.CONTAINER,
          groupId: "motion",
          presentationTimestampUs: 123456n,
          hasEmbeddedVideo: true,
        },
      }),
    ]);

    expect(item.kind).toBe("motion");
    expect(item.posterUrl).toBe(`${window.location.origin}/file/attachments/motion/motion.jpg?thumbnail=true`);
    expect(item.sourceUrl).toBe("https://media.example.com/assets/motion.jpg");
    expect(item.previewItem).toMatchObject({
      kind: "motion",
      posterUrl: `${window.location.origin}/file/attachments/motion/motion.jpg?thumbnail=true`,
      motionUrl: `${window.location.origin}/file/attachments/motion/motion.jpg?motion=true`,
    });
  });

  it("uses the still thumbnail and motion video original for Apple Live Photo pairs", () => {
    const [item] = buildAttachmentVisualItems([
      attachment({
        name: "attachments/still",
        filename: "still.jpg",
        externalLink: "https://media.example.com/assets/still.jpg",
        motionMedia: {
          family: MotionMediaFamily.APPLE_LIVE_PHOTO,
          role: MotionMediaRole.STILL,
          groupId: "live-photo",
          presentationTimestampUs: 0n,
          hasEmbeddedVideo: false,
        },
      }),
      attachment({
        name: "attachments/video",
        filename: "video.mov",
        type: "video/quicktime",
        externalLink: "https://media.example.com/assets/video.mov",
        motionMedia: {
          family: MotionMediaFamily.APPLE_LIVE_PHOTO,
          role: MotionMediaRole.VIDEO,
          groupId: "live-photo",
          presentationTimestampUs: 0n,
          hasEmbeddedVideo: false,
        },
      }),
    ]);

    expect(item.kind).toBe("motion");
    expect(item.posterUrl).toBe(`${window.location.origin}/file/attachments/still/still.jpg?thumbnail=true`);
    expect(item.sourceUrl).toBe("https://media.example.com/assets/video.mov");
    expect(item.previewItem).toMatchObject({
      kind: "motion",
      posterUrl: `${window.location.origin}/file/attachments/still/still.jpg?thumbnail=true`,
      motionUrl: "https://media.example.com/assets/video.mov",
    });
  });
});
