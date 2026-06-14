import { describe, expect, it } from "vitest";
import { toAttachmentItems } from "@/components/MemoEditor/types/attachment";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { MotionMediaFamily, MotionMediaRole } from "@/types/proto/api/v1/attachment_service_pb";

const attachment = (overrides: Partial<Attachment>): Attachment =>
  ({
    name: "attachments/test",
    filename: "test.jpg",
    type: "image/jpeg",
    externalLink: "https://media.example.com/assets/test.jpg",
    ...overrides,
  }) as Attachment;

describe("toAttachmentItems", () => {
  it("keeps Android Motion Photo originals separate from extracted motion clips", () => {
    const [item] = toAttachmentItems([
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

    expect(item.category).toBe("motion");
    expect(item.sourceUrl).toBe("https://media.example.com/assets/motion.jpg");
    expect(item.thumbnailUrl).toBe(`${window.location.origin}/file/attachments/motion/motion.jpg?thumbnail=true`);
    expect(item.motionUrl).toBe(`${window.location.origin}/file/attachments/motion/motion.jpg?motion=true`);
  });

  it("uses the paired video as the Apple Live Photo motion source", () => {
    const [item] = toAttachmentItems([
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

    expect(item.category).toBe("motion");
    expect(item.sourceUrl).toBe("https://media.example.com/assets/video.mov");
    expect(item.thumbnailUrl).toBe(`${window.location.origin}/file/attachments/still/still.jpg?thumbnail=true`);
    expect(item.motionUrl).toBe("https://media.example.com/assets/video.mov");
  });
});
