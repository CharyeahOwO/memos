import { hasUploadingLocalFiles, type LocalFile } from "../types/attachment";
import type { EditorAction, EditorState } from "./types";
import { initialState } from "./types";

function withLocalFiles(state: EditorState, localFiles: LocalFile[]): EditorState {
  return {
    ...state,
    localFiles,
    ui: {
      ...state.ui,
      isLoading: {
        ...state.ui.isLoading,
        uploading: hasUploadingLocalFiles(localFiles),
      },
    },
  };
}

export function editorReducer(state: EditorState, action: EditorAction): EditorState {
  switch (action.type) {
    case "INIT_MEMO":
      return {
        ...state,
        content: action.payload.content,
        metadata: action.payload.metadata,
        timestamps: action.payload.timestamps,
      };

    case "UPDATE_CONTENT":
      return {
        ...state,
        content: action.payload,
      };

    case "SET_METADATA":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          ...action.payload,
        },
      };

    case "ADD_ATTACHMENT":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          attachments: [...state.metadata.attachments, action.payload],
        },
      };

    case "REMOVE_ATTACHMENT":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          attachments: state.metadata.attachments.filter((a) => a.name !== action.payload),
        },
      };

    case "ADD_RELATION":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          relations: [...state.metadata.relations, action.payload],
        },
      };

    case "REMOVE_RELATION":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          relations: state.metadata.relations.filter((r) => r.relatedMemo?.name !== action.payload),
        },
      };

    case "ADD_LOCAL_FILE":
      return withLocalFiles(state, [...state.localFiles, action.payload]);

    case "REMOVE_LOCAL_FILE":
      return withLocalFiles(
        state,
        state.localFiles.filter((f) => f.previewUrl !== action.payload),
      );

    case "SET_LOCAL_FILES":
      return withLocalFiles(state, action.payload);

    case "UPDATE_LOCAL_FILE_UPLOAD":
      return withLocalFiles(
        state,
        state.localFiles.map((localFile) => {
          if (localFile.previewUrl !== action.payload.previewUrl) {
            return localFile;
          }

          return {
            ...localFile,
            uploadStatus: action.payload.status,
            uploadError: action.payload.error,
            uploadProgress: action.payload.progress,
          };
        }),
      );

    case "CLEAR_LOCAL_FILES":
      return withLocalFiles(state, []);

    case "TOGGLE_FOCUS_MODE":
      return {
        ...state,
        ui: {
          ...state.ui,
          isFocusMode: !state.ui.isFocusMode,
        },
      };

    case "SET_LOADING":
      return {
        ...state,
        ui: {
          ...state.ui,
          isLoading: {
            ...state.ui.isLoading,
            [action.payload.key]: action.payload.value,
          },
        },
      };

    case "SET_COMPOSING":
      return {
        ...state,
        ui: {
          ...state.ui,
          isComposing: action.payload,
        },
      };

    case "SET_TIMESTAMPS":
      return {
        ...state,
        timestamps: {
          ...state.timestamps,
          ...action.payload,
        },
      };

    case "SET_AUDIO_RECORDER_SUPPORT":
      return {
        ...state,
        audioRecorder: {
          ...state.audioRecorder,
          isSupported: action.payload,
          status: action.payload ? state.audioRecorder.status : "unsupported",
        },
      };

    case "SET_AUDIO_RECORDER_PERMISSION":
      return {
        ...state,
        audioRecorder: {
          ...state.audioRecorder,
          permission: action.payload,
        },
      };

    case "SET_AUDIO_RECORDER_STATUS":
      return {
        ...state,
        audioRecorder: {
          ...state.audioRecorder,
          status: action.payload,
        },
      };

    case "SET_AUDIO_RECORDER_ELAPSED":
      return {
        ...state,
        audioRecorder: {
          ...state.audioRecorder,
          elapsedSeconds: action.payload,
        },
      };

    case "SET_AUDIO_RECORDER_ERROR":
      return {
        ...state,
        audioRecorder: {
          ...state.audioRecorder,
          error: action.payload,
        },
      };

    case "RESET":
      return {
        ...initialState,
      };

    default:
      return state;
  }
}
