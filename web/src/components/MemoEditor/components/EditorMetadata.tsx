import type { FC } from "react";
import { AttachmentListEditor, LocationDisplayEditor, RelationListEditor } from "@/components/MemoMetadata";
import { useEditorContext } from "../state";
import type { EditorMetadataProps } from "../types";

export const EditorMetadata: FC<EditorMetadataProps> = ({ memoName, onLocalFilesChange, onRemoveLocalFile, onRetryLocalFile }) => {
  const { state, actions, dispatch } = useEditorContext();

  return (
    <div className="w-full flex flex-col gap-2">
      <AttachmentListEditor
        attachments={state.metadata.attachments}
        localFiles={state.localFiles}
        onAttachmentsChange={(attachments) => dispatch(actions.setMetadata({ attachments }))}
        onLocalFilesChange={onLocalFilesChange}
        onRemoveLocalFile={onRemoveLocalFile}
        onRetryLocalFile={onRetryLocalFile}
      />

      <RelationListEditor
        relations={state.metadata.relations}
        onRelationsChange={(relations) => dispatch(actions.setMetadata({ relations }))}
        memoName={memoName}
      />

      {state.metadata.location && (
        <LocationDisplayEditor location={state.metadata.location} onRemove={() => dispatch(actions.setMetadata({ location: undefined }))} />
      )}
    </div>
  );
};
