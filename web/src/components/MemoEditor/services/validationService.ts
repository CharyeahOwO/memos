import type { EditorState } from "../state";
import { hasFailedLocalFiles } from "../types/attachment";

export interface ValidationResult {
  valid: boolean;
  reason?: string;
}

export const validationService = {
  canSave(state: EditorState): ValidationResult {
    // Cannot save while loading initial content
    if (state.ui.isLoading.loading) {
      return { valid: false, reason: "Loading memo content" };
    }

    // Cannot save while local attachments are still uploading.
    if (state.ui.isLoading.uploading) {
      return { valid: false, reason: "Wait for upload to complete" };
    }

    // Failed local uploads must be removed or retried before the memo can reference attachments.
    if (hasFailedLocalFiles(state.localFiles)) {
      return { valid: false, reason: "Remove failed uploads before saving" };
    }

    // Any remaining local file has not been converted to a server-side attachment yet.
    if (state.localFiles.length > 0) {
      return { valid: false, reason: "Wait for all attachments to finish uploading before saving" };
    }

    // Must have content or uploaded attachment.
    if (!state.content.trim() && state.metadata.attachments.length === 0) {
      return { valid: false, reason: "Content or uploaded attachment required" };
    }

    // Cannot save while audio recorder is active
    if (state.audioRecorder.status === "recording" || state.audioRecorder.status === "requesting_permission") {
      return { valid: false, reason: "Finish audio recording before saving" };
    }

    // Cannot save while already saving
    if (state.ui.isLoading.saving) {
      return { valid: false, reason: "Save in progress" };
    }

    return { valid: true };
  },
};
