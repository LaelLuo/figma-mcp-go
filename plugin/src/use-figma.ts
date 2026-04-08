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

  const data = await runner(figma, request);
  return {
    type: request.type,
    requestId: request.requestId,
    data,
  };
};
