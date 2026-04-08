import { beforeEach, describe, expect, it } from "bun:test";
import { handleUseFigmaRequest } from "./use-figma";

const makeRequest = (code: string) => ({
  type: "use_figma",
  requestId: "req-use-figma-1",
  params: {
    code,
    description: "test request",
  },
});

beforeEach(() => {
  (globalThis as any).figma = {
    root: { name: "Fallback File" },
    currentPage: { id: "0:1", name: "Page 1" },
  };
});

describe("handleUseFigmaRequest", () => {
  it("runs async code and returns the result", async () => {
    const response = await handleUseFigmaRequest(
      makeRequest(`
        const fileName = await Promise.resolve(figma.root.name);
        return { fileName };
      `),
    );

    expect(response?.type).toBe("use_figma");
    expect(response?.requestId).toBe("req-use-figma-1");
    expect(response?.data).toEqual({ fileName: "Fallback File" });
  });

  it("surfaces thrown errors with message text", async () => {
    await expect(
      handleUseFigmaRequest(
        makeRequest(`
          throw new Error("boom");
        `),
      ),
    ).rejects.toThrow("boom");
  });

  it("supports top-level await semantics via async wrapper", async () => {
    const response = await handleUseFigmaRequest(
      makeRequest(`
        const pageName = await Promise.resolve(figma.currentPage.name);
        return pageName;
      `),
    );

    expect(response?.data).toBe("Page 1");
  });

  it("returns undefined cleanly when code does not explicitly return", async () => {
    const response = await handleUseFigmaRequest(
      makeRequest(`
        await Promise.resolve("done");
      `),
    );

    expect(response?.data).toBeUndefined();
  });
});
