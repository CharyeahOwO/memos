import { useCallback, useEffect, useRef } from "react";
import { toast } from "react-hot-toast";
import { useTranslate } from "@/utils/i18n";
import { errorService, uploadService } from "../services";
import { useEditorContext } from "../state";
import type { LocalFile } from "../types/attachment";

const toQueuedLocalFile = (localFile: LocalFile): LocalFile => ({
  ...localFile,
  uploadStatus: "pending",
  uploadError: undefined,
  uploadProgress: 0,
});

export function useAttachmentUploadQueue() {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();
  const localFileIdsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    localFileIdsRef.current = new Set(state.localFiles.map((localFile) => localFile.previewUrl));
  }, [state.localFiles]);

  const removeLocalFile = useCallback(
    (previewUrl: string) => {
      localFileIdsRef.current.delete(previewUrl);
      dispatch(actions.removeLocalFile(previewUrl));
    },
    [actions, dispatch],
  );

  const setLocalFiles = useCallback(
    (localFiles: LocalFile[]) => {
      localFileIdsRef.current = new Set(localFiles.map((localFile) => localFile.previewUrl));
      dispatch(actions.setLocalFiles(localFiles));
    },
    [actions, dispatch],
  );

  const uploadLocalFile = useCallback(
    async (localFile: LocalFile) => {
      const previewUrl = localFile.previewUrl;
      localFileIdsRef.current.add(previewUrl);
      dispatch(actions.updateLocalFileUpload(previewUrl, { status: "uploading", error: undefined, progress: undefined }));

      try {
        const attachment = await uploadService.uploadFile(localFile);
        if (!localFileIdsRef.current.has(previewUrl)) {
          return;
        }

        dispatch(actions.addAttachment(attachment));
        localFileIdsRef.current.delete(previewUrl);
        dispatch(actions.removeLocalFile(previewUrl));
      } catch (error) {
        if (!localFileIdsRef.current.has(previewUrl)) {
          return;
        }

        const message = errorService.getErrorMessage(error) || t("editor.attachment-upload.failed");
        dispatch(actions.updateLocalFileUpload(previewUrl, { status: "failed", error: message, progress: undefined }));
        toast.error(message);
      }
    },
    [actions, dispatch, t],
  );

  const uploadLocalFiles = useCallback(
    (localFiles: LocalFile[]) => {
      if (localFiles.length === 0) {
        return;
      }

      const queuedLocalFiles = localFiles.map(toQueuedLocalFile);
      for (const localFile of queuedLocalFiles) {
        localFileIdsRef.current.add(localFile.previewUrl);
        dispatch(actions.addLocalFile(localFile));
      }

      void (async () => {
        for (const localFile of queuedLocalFiles) {
          if (localFileIdsRef.current.has(localFile.previewUrl)) {
            await uploadLocalFile(localFile);
          }
        }
      })();
    },
    [actions, dispatch, uploadLocalFile],
  );

  const retryLocalFile = useCallback(
    (previewUrl: string) => {
      const localFile = state.localFiles.find((file) => file.previewUrl === previewUrl);
      if (!localFile) {
        return;
      }

      void uploadLocalFile(toQueuedLocalFile(localFile));
    },
    [state.localFiles, uploadLocalFile],
  );

  return {
    uploadLocalFiles,
    retryLocalFile,
    setLocalFiles,
    removeLocalFile,
  };
}
