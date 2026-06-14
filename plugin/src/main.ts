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

// Per-request timeout (ms). A hung handler (e.g. exportAsync that never resolves)
// must not block the serial queue forever — otherwise every later request stalls
// behind it and the only recovery is manually re-running the plugin. Values sit a
// few seconds ABOVE the Go bridge timeouts (use_figma/get_document = 90s, others
// 30s) so the server reports the timeout first and the queue then auto-recovers.
const LONG_REQUEST_TYPES = new Set([
  "get_document",
  "get_design_context",
  "get_figjam",
  "use_figma",
]);
const requestTimeoutMs = (type: string) =>
  LONG_REQUEST_TYPES.has(type) ? 95000 : 35000;

const withTimeout = <T>(promise: Promise<T>, ms: number, type: string): Promise<T> =>
  new Promise<T>((resolve, reject) => {
    const timer = setTimeout(
      () =>
        reject(
          new Error(
            `plugin request '${type}' timed out after ${ms}ms (queue auto-recovered; a stuck handler was abandoned)`,
          ),
        ),
      ms,
    );
    promise.then(
      (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      (error) => {
        clearTimeout(timer);
        reject(error);
      },
    );
  });

const enqueueRequest = (request: any) => {
  requestQueue = requestQueue
    .catch(() => {
      // Keep the queue alive after earlier request failures.
    })
    .then(async () => {
      postProgress(request.requestId, 1, `accepted ${request.type}`);
      // handleRequest already swallows handler errors into an error response, so the
      // only way this rejects is the timeout — which frees the queue for the next request.
      let response: any;
      try {
        response = await withTimeout(
          handleRequest(request),
          requestTimeoutMs(request.type),
          request.type,
        );
      } catch (err) {
        response = {
          type: request.type,
          requestId: request.requestId,
          error: err instanceof Error ? err.message : String(err),
        };
      }
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
