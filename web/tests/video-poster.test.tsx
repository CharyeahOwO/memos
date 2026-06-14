import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import VideoPoster from "@/components/VideoPoster";

describe("<VideoPoster>", () => {
  it("renders a lightweight placeholder without loading the video by default", () => {
    render(<VideoPoster sourceUrl="/file/attachments/video/video.mp4" alt="clip.mp4" className="object-cover" />);

    expect(screen.getByTestId("video-poster-placeholder")).toHaveAttribute("aria-label", "clip.mp4");
    expect(screen.queryByTestId("video-poster-fallback")).not.toBeInTheDocument();
  });

  it("renders a mobile-friendly video fallback when frame capture is enabled", () => {
    render(<VideoPoster sourceUrl="/file/attachments/video/video.mp4" alt="clip.mp4" className="object-cover" captureFrame />);

    const video = screen.getByTestId("video-poster-fallback");
    expect(video).toHaveAttribute("preload", "metadata");
    expect(video).toHaveAttribute("playsinline");
    expect((video as HTMLVideoElement).muted).toBe(true);
    expect(video).toHaveClass("object-cover");
  });

  it("renders a captured frame as an image poster", async () => {
    const drawImage = vi.fn();
    const toDataURL = vi.fn(() => "data:image/jpeg;base64,poster");
    const originalCreateElement = document.createElement.bind(document);

    vi.spyOn(document, "createElement").mockImplementation((tagName: string, options?: ElementCreationOptions) => {
      if (tagName === "canvas") {
        return {
          width: 0,
          height: 0,
          getContext: vi.fn(() => ({ drawImage })),
          toDataURL,
        } as unknown as HTMLCanvasElement;
      }

      return originalCreateElement(tagName, options);
    });

    render(<VideoPoster sourceUrl="/file/attachments/video/video.mp4" alt="clip.mp4" className="object-cover" captureFrame />);

    const video = screen.getByTestId("video-poster-fallback") as HTMLVideoElement;
    Object.defineProperty(video, "videoWidth", { configurable: true, value: 640 });
    Object.defineProperty(video, "videoHeight", { configurable: true, value: 360 });

    fireEvent.loadedData(video);

    await waitFor(() => {
      expect(screen.getByRole("img", { name: "clip.mp4" })).toHaveAttribute("src", "data:image/jpeg;base64,poster");
    });
    expect(drawImage).toHaveBeenCalledWith(video, 0, 0, 640, 360);
  });
});
