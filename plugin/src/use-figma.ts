const HEARTBEAT_INTERVAL_MS = 2000;

const startHeartbeat = (requestId: string) =>
  setInterval(() => {
    figma.ui.postMessage({
      type: "progress_update",
      requestId,
      progress: 5,
      message: "running use_figma",
    });
  }, HEARTBEAT_INTERVAL_MS);

export const handleUseFigmaRequest = async (request: any) => {
  if (request.type !== "use_figma") {
    return null;
  }

  const code = request.params?.code;
  if (typeof code !== "string" || code.trim() === "") {
    throw new Error("code is required");
  }

  const runner = new Function(
    "figma",
    "request",
    `"use strict"; return (async () => { ${code} })();`,
  ) as (figma: PluginAPI, request: any) => Promise<unknown>;

  const heartbeat = startHeartbeat(request.requestId);
  try {
    const data = await runner(figma, request);
    return {
      type: request.type,
      requestId: request.requestId,
      data,
    };
  } finally {
    clearInterval(heartbeat);
  }
};
