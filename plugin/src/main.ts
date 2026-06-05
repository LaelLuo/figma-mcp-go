// Plugin core — entry point, UI bootstrap, and request dispatch.

import { handleReadRequest } from "./read-handlers";
import { handleUseFigmaRequest } from "./use-figma";
import { handleWriteRequest } from "./write-handlers";

const sendStatus = () => {
  figma.ui.postMessage({
    type: "plugin-status",
    payload: {
      fileName: figma.root.name,
      pageName: figma.currentPage.name,
      selectionCount: figma.currentPage.selection.length,
    },
  });
};

const handleRequest = async (request: any) => {
  try {
    const result =
      (await handleUseFigmaRequest(request)) ??
      (await handleReadRequest(request)) ??
      (await handleWriteRequest(request));
    if (result === null)
      throw new Error(`Unknown request type: ${request.type}`);
    return result;
  } catch (error) {
    return {
      type: request.type,
      requestId: request.requestId,
      error: error instanceof Error ? error.message : String(error),
    };
  }
};

const postProgress = (requestId: string, progress: number, message: string) => {
  figma.ui.postMessage({
    type: "progress_update",
    requestId,
    progress,
    message,
  });
};

let requestQueue: Promise<void> = Promise.resolve();

const enqueueRequest = (request: any) => {
  requestQueue = requestQueue
    .catch(() => {
      // Keep the queue alive after earlier request failures.
    })
    .then(async () => {
      postProgress(request.requestId, 1, `accepted ${request.type}`);
      const response = await handleRequest(request);
      try {
        figma.ui.postMessage(response);
      } catch (err) {
        figma.ui.postMessage({
          type: response.type,
          requestId: response.requestId,
          error: err instanceof Error ? err.message : String(err),
        });
      }
    });
};

figma.showUI(__html__, { width: 320, height: 210 });
sendStatus();

figma.on("selectionchange", () => {
  sendStatus();
});

figma.on("currentpagechange", () => {
  sendStatus();
});

figma.ui.onmessage = async (message) => {
  if (message.type === "ui-ready") {
    sendStatus();
    return;
  }
  if (message.type === "server-request") {
    enqueueRequest(message.payload);
  }
};
