import { PlayIcon } from "lucide-react";
import VideoPoster from "@/components/VideoPoster";
import { cn } from "@/lib/utils";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { getAttachmentThumbnailUrl, getAttachmentType, getAttachmentUrl } from "@/utils/attachment";

interface AttachmentCardProps {
  attachment: Attachment;
  onClick?: () => void;
  className?: string;
}

const AttachmentCard = ({ attachment, onClick, className }: AttachmentCardProps) => {
  const attachmentType = getAttachmentType(attachment);
  const sourceUrl = getAttachmentUrl(attachment);

  if (attachmentType === "image/*") {
    return (
      <img
        src={getAttachmentThumbnailUrl(attachment)}
        alt={attachment.filename}
        className={cn("w-full h-full object-cover rounded-lg cursor-pointer", className)}
        onClick={onClick}
        onError={(e) => {
          const target = e.currentTarget;
          if (target.src.includes("?thumbnail=true")) {
            target.src = sourceUrl;
          }
        }}
        decoding="async"
        loading="lazy"
      />
    );
  }

  if (attachmentType === "video/*") {
    return (
      <button
        type="button"
        className={cn("relative block overflow-hidden rounded-lg bg-muted/40", className)}
        onClick={onClick ?? (() => window.open(sourceUrl))}
        aria-label={`Open ${attachment.filename}`}
      >
        <VideoPoster sourceUrl={sourceUrl} alt={attachment.filename} className="h-full w-full object-cover" />
        <span className="absolute inset-0 flex items-center justify-center text-foreground">
          <span className="inline-flex h-9 w-9 items-center justify-center rounded-full bg-background/85 shadow-sm">
            <PlayIcon className="h-4 w-4 fill-current" />
          </span>
        </span>
      </button>
    );
  }

  if (attachmentType === "audio/*") {
    return <audio src={sourceUrl} className={cn("w-full rounded-lg", className)} controls preload="metadata" />;
  }

  return null;
};

export default AttachmentCard;
