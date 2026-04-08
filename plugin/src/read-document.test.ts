import { beforeEach, describe, expect, it } from "bun:test";
import { handleReadDocumentRequest } from "./read-document";

const makeRequest = (type: string, params?: any) => ({
  type,
  requestId: "req-doc-1",
  nodeIds: [],
  params: params ?? {},
});

const toBase64 = (bytes: Uint8Array) => {
  const alphabet =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  let result = "";
  for (let i = 0; i < bytes.length; i += 3) {
    const a = bytes[i] ?? 0;
    const b = bytes[i + 1] ?? 0;
    const c = bytes[i + 2] ?? 0;
    const chunk = (a << 16) | (b << 8) | c;
    result += alphabet[(chunk >> 18) & 63];
    result += alphabet[(chunk >> 12) & 63];
    result += i + 1 < bytes.length ? alphabet[(chunk >> 6) & 63] : "=";
    result += i + 2 < bytes.length ? alphabet[chunk & 63] : "=";
  }
  return result;
};

const buildMockDocument = () => {
  const textNode = {
    id: "2:1",
    name: "Title",
    type: "TEXT",
    x: 16,
    y: 20,
    width: 120,
    height: 24,
    characters: "Hello fallback",
    visible: true,
    opacity: 1,
    fills: [],
    strokes: [],
    fillStyleId: "",
    strokeStyleId: "",
    fontName: { family: "Inter", style: "Regular" },
    fontSize: 16,
    fontWeight: 400,
    textDecoration: "NONE",
    lineHeight: { unit: "AUTO" },
    letterSpacing: { value: 0, unit: "PIXELS" },
    textAlignHorizontal: "LEFT",
  };

  const frameNode = {
    id: "1:1",
    name: "Card",
    type: "FRAME",
    x: 0,
    y: 0,
    width: 240,
    height: 120,
    visible: true,
    opacity: 1,
    fills: [],
    strokes: [],
    fillStyleId: "",
    strokeStyleId: "",
    cornerRadius: 12,
    paddingTop: 16,
    paddingRight: 16,
    paddingBottom: 16,
    paddingLeft: 16,
    children: [textNode],
    exportAsync: async () => new Uint8Array([1, 2, 3, 4]),
  };

  const pageNode = {
    id: "0:1",
    name: "Page 1",
    type: "PAGE",
    selection: [frameNode],
    children: [frameNode],
    loadAsync: async () => {},
  };

  const nodes = new Map<string, any>([
    [pageNode.id, pageNode],
    [frameNode.id, frameNode],
    [textNode.id, textNode],
  ]);

  return {
    root: {
      name: "Fallback Test File",
      children: [pageNode],
    },
    currentPage: pageNode,
    nodes,
  };
};

beforeEach(() => {
  const doc = buildMockDocument();
  (globalThis as any).figma = {
    editorType: "figma",
    root: doc.root,
    currentPage: doc.currentPage,
    viewport: {
      center: { x: 0, y: 0 },
      zoom: 1,
      bounds: { x: 0, y: 0, width: 800, height: 600 },
    },
    getNodeByIdAsync: async (id: string) => doc.nodes.get(id) ?? null,
    getStyleByIdAsync: async () => null,
    base64Encode: (bytes: Uint8Array) => toBase64(bytes),
  };
});

describe("read-document fallback envelopes", () => {
  it("get_metadata returns fallback envelope with xml-ish metadataText", async () => {
    const res = await handleReadDocumentRequest(makeRequest("get_metadata"));

    expect(res?.data.fileKey).toBeNull();
    expect(res?.data.nodeId).toBe("1:1");
    expect(res?.data.kind).toBe("selection");
    expect(res?.data.pages).toHaveLength(1);
    expect(typeof res?.data.metadataText).toBe("string");
    expect(res?.data.metadataText).toContain("<metadata");
    expect(res?.data.metadataText).toContain('source="figma-mcp-go"');
    expect(res?.data.metadataText).toContain('id="1:1"');
    expect(res?.data.metadataText).toContain('type="FRAME"');
  });

  it("get_design_context returns fallback envelope without screenshot when excludeScreenshot=true", async () => {
    const res = await handleReadDocumentRequest(
      makeRequest("get_design_context", { excludeScreenshot: true, detail: "minimal" }),
    );

    expect(res?.data.fileKey).toBeNull();
    expect(res?.data.nodeId).toBe("1:1");
    expect(res?.data.name).toBe("Card");
    expect(typeof res?.data.code).toBe("string");
    expect(res?.data.context).toBeDefined();
    expect(res?.data.metadata?.source).toBe("figma-mcp-go");
    expect(typeof res?.data.metadata?.degraded).toBe("boolean");
    expect("screenshot" in (res?.data ?? {})).toBe(false);
  });

  it("get_design_context returns a stable screenshot envelope by default", async () => {
    const res = await handleReadDocumentRequest(makeRequest("get_design_context"));

    expect(res?.data.screenshot).toBeDefined();
    expect(res?.data.screenshot.nodeId).toBe("1:1");
    expect(res?.data.screenshot.mimeType).toBe("image/png");
    expect(res?.data.screenshot.base64).toBe("AQIDBA==");
  });

  it("get_figjam throws on non-figjam files", async () => {
    await expect(
      handleReadDocumentRequest(makeRequest("get_figjam", { nodeId: "0:1" })),
    ).rejects.toThrow("get_figjam is only available in FigJam files");
  });
});
