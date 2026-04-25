import { serializeNode, getBounds, serializeStyles, isMixed, deduplicateStyles } from "./serializers";

const FALLBACK_SOURCE = "figma-mcp-go";
const MAX_DESIGN_CONTEXT_CODE_LENGTH = 120000;

const escapeXml = (value: any) =>
  String(value)
    .replace(/&/g, "&amp;")
    .replace(/"/g, "&quot;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

const getPageSummaries = () =>
  figma.root.children.map((page) => ({
    id: page.id,
    name: page.name,
    current: page.id === figma.currentPage.id,
  }));

const getNodeById = async (nodeId: string) => {
  const node = await figma.getNodeByIdAsync(nodeId);
  if (!node || node.type === "DOCUMENT") {
    throw new Error(`Node not found: ${nodeId}`);
  }
  return node;
};

const resolveTargetNodes = async (request: any, includeCurrentPage = true) => {
  const nodeId = request.params && request.params.nodeId;
  if (nodeId) {
    return [await getNodeById(nodeId)];
  }
  const selection = figma.currentPage.selection ?? [];
  if (selection.length > 0) {
    return selection;
  }
  return includeCurrentPage ? [figma.currentPage] : [];
};

const serializeSparseNodeLine = (node: any) => {
  const attrs = [
    `id="${escapeXml(node.id)}"`,
    `name="${escapeXml(node.name ?? "")}"`,
    `type="${escapeXml(node.type)}"`,
  ];
  const bounds = getBounds(node);
  if (bounds) {
    attrs.push(
      `x="${bounds.x}"`,
      `y="${bounds.y}"`,
      `width="${bounds.width}"`,
      `height="${bounds.height}"`,
    );
  }
  if ("children" in node && Array.isArray(node.children)) {
    attrs.push(`childCount="${node.children.length}"`);
  }
  return `    <node ${attrs.join(" ")} />`;
};

const buildMetadataText = (kind: string, nodes: readonly any[]) => {
  const pages = getPageSummaries();
  return [
    `<metadata source="${FALLBACK_SOURCE}" degraded="true" kind="${escapeXml(kind)}" selectionCount="${nodes.length}">`,
    "  <pages>",
    ...pages.map(
      (page) =>
        `    <page id="${escapeXml(page.id)}" name="${escapeXml(page.name)}" current="${page.current}" />`,
    ),
    "  </pages>",
    "  <selection>",
    ...nodes.map((node) => serializeSparseNodeLine(node)),
    "  </selection>",
    "</metadata>",
  ].join("\n");
};

const buildFigJamMetadataText = (nodes: readonly any[]) => [
  `<figjam source="${FALLBACK_SOURCE}" degraded="true" selectionCount="${nodes.length}">`,
  "  <nodes>",
  ...nodes.map((node) => serializeSparseNodeLine(node)),
  "  </nodes>",
  "</figjam>",
].join("\n");

const countSerializedNodes = (value: any): number => {
  if (!value || typeof value !== "object") return 0;
  if (Array.isArray(value)) return value.reduce((sum, item) => sum + countSerializedNodes(item), 0);
  const childCount = Array.isArray(value.children) ? countSerializedNodes(value.children) : 0;
  return ("id" in value && "type" in value ? 1 : 0) + childCount;
};

const exportNodeScreenshot = async (node: any) => {
  const screenshot: any = {
    nodeId: node.id,
    mimeType: "image/png",
    base64: null,
  };
  if ("width" in node) screenshot.width = node.width;
  if ("height" in node) screenshot.height = node.height;
  if (typeof node.exportAsync !== "function") {
    screenshot.error = "PNG export is not available for this node in local fallback mode.";
    return screenshot;
  }
  try {
    const bytes = await node.exportAsync({
      format: "PNG",
      constraint: { type: "SCALE", value: 2 },
    });
    screenshot.base64 = figma.base64Encode(bytes);
    return screenshot;
  } catch (error) {
    screenshot.error = error instanceof Error ? error.message : String(error);
    return screenshot;
  }
};

export const handleReadDocumentRequest = async (request: any) => {
  switch (request.type) {
    case "get_document": {
      const raw = await serializeNode(figma.currentPage);
      const { tree, globalVars } = deduplicateStyles(raw);
      return {
        type: request.type,
        requestId: request.requestId,
        data: globalVars ? { ...tree, globalVars } : tree,
      };
    }

    case "get_selection":
      return {
        type: request.type,
        requestId: request.requestId,
        data: await Promise.all(figma.currentPage.selection.map((node) => serializeNode(node))),
      };

    case "get_node": {
      const nodeId = request.nodeIds && request.nodeIds[0];
      if (!nodeId) throw new Error("nodeIds is required for get_node");
      const node = await figma.getNodeByIdAsync(nodeId);
      if (!node || node.type === "DOCUMENT")
        throw new Error(`Node not found: ${nodeId}`);
      if (node.type === "PAGE") {
        throw new Error(
          "get_node does not support PAGE nodes because it serializes the whole page. Use get_metadata or get_design_context with depth=1 instead.",
        );
      }
      return {
        type: request.type,
        requestId: request.requestId,
        data: await serializeNode(node),
      };
    }

    case "get_nodes_info": {
      if (!request.nodeIds || request.nodeIds.length === 0)
        throw new Error("nodeIds is required for get_nodes_info");
      const nodes = await Promise.all(
        request.nodeIds.map((id: string) => figma.getNodeByIdAsync(id)),
      );
      return {
        type: request.type,
        requestId: request.requestId,
        data: await Promise.all(
          nodes
            .filter((n) => n !== null && n.type !== "DOCUMENT")
            .map((n) => serializeNode(n)),
        ),
      };
    }

    case "get_design_context": {
      const depth =
        request.params && request.params.depth != null
          ? request.params.depth
          : 2;
      const detail = (request.params && request.params.detail) || "full";
      const dedupeComponents = !!(request.params && request.params.dedupeComponents);
      const componentDefs = new Map<string, any>();

      const serializeForDetail = async (n: any) => {
        const base = { id: n.id, name: n.name, type: n.type, bounds: getBounds(n) };
        if (detail === "minimal") return base;
        const styles = await serializeStyles(n);
        const result: any = Object.assign({}, base);
        if (Object.keys(styles).length > 0) result.styles = styles;
        if ("opacity" in n && n.opacity !== 1) result.opacity = n.opacity;
        if ("visible" in n && !n.visible) result.visible = false;
        if (detail === "compact") return result;
        return await serializeNode(n);
      };

      const extractInstanceOverrides = async (
        instanceNode: any,
        componentNode: any,
      ): Promise<{ id: string; name: string; type: string; characters?: string; mainComponentId?: string | null; visible?: boolean; opacity?: number; fills?: any }[]> => {
        const overrides: any[] = [];
        if (!instanceNode?.children || !componentNode?.children) return overrides;
        for (let i = 0; i < instanceNode.children.length; i++) {
          const instChild = instanceNode.children[i];
          const compChild = componentNode.children[i];
          if (!instChild || !compChild) continue;

          // Detect property overrides (visible, opacity, fills) for all node types
          const propChanges: any = {};
          if ("visible" in instChild && "visible" in compChild && instChild.visible !== compChild.visible) {
            propChanges.visible = instChild.visible;
          }
          if ("opacity" in instChild && "opacity" in compChild && instChild.opacity !== compChild.opacity) {
            propChanges.opacity = instChild.opacity;
          }
          if ("fills" in instChild && "fills" in compChild && !isMixed(instChild.fills) && !isMixed(compChild.fills)) {
            if (JSON.stringify(instChild.fills) !== JSON.stringify(compChild.fills)) {
              propChanges.fills = instChild.fills;
            }
          }

          if (instChild.type === "TEXT") {
            const override: any = { id: instChild.id, name: instChild.name, type: "TEXT" };
            let hasChange = false;
            if (instChild.characters !== compChild.characters) {
              override.characters = instChild.characters;
              hasChange = true;
            }
            if (Object.keys(propChanges).length > 0) {
              Object.assign(override, propChanges);
              hasChange = true;
            }
            if (hasChange) overrides.push(override);
            continue;
          }

          if (instChild.type === "INSTANCE") {
            const [nestedMc, compMc] = await Promise.all([
              instChild.getMainComponentAsync(),
              compChild.type === "INSTANCE" ? compChild.getMainComponentAsync() : Promise.resolve(null),
            ]);
            if (nestedMc?.id !== compMc?.id) {
              const override: any = { id: instChild.id, name: instChild.name, type: "INSTANCE", mainComponentId: nestedMc?.id ?? null };
              if (Object.keys(propChanges).length > 0) Object.assign(override, propChanges);
              overrides.push(override);
              continue;
            }
            if (Object.keys(propChanges).length > 0) {
              overrides.push({ id: instChild.id, name: instChild.name, type: "INSTANCE", mainComponentId: nestedMc?.id ?? null, ...propChanges });
            }
            if (nestedMc) overrides.push(...await extractInstanceOverrides(instChild, nestedMc));
            continue;
          }

          if (Object.keys(propChanges).length > 0) {
            overrides.push({ id: instChild.id, name: instChild.name, type: instChild.type, ...propChanges });
          }
          if ("children" in instChild) {
            overrides.push(...await extractInstanceOverrides(instChild, compChild));
          }
        }
        return overrides;
      };

      const serializeWithDepth = async (node: any, currentDepth: number): Promise<any> => {
        if (dedupeComponents && node.type === "INSTANCE") {
          const mc = await node.getMainComponentAsync();
          if (mc && !componentDefs.has(mc.id)) {
            componentDefs.set(mc.id, await serializeNode(mc));
          }
          const props: Record<string, any> = {};
          if (node.componentProperties) {
            for (const [key, prop] of Object.entries(node.componentProperties)) {
              props[key] = (prop as any).value;
            }
          }
          const result: any = {
            id: node.id,
            name: node.name,
            type: node.type,
            bounds: getBounds(node),
            mainComponentId: mc?.id ?? null,
          };
          if (Object.keys(props).length > 0) result.componentProperties = props;
          const overrides = await extractInstanceOverrides(node, mc);
          if (overrides.length > 0) result.overrides = overrides;
          return result;
        }
        if (detail === "full") {
          const serialized = await serializeNode(node);
          if (currentDepth >= depth && serialized.children) {
            return Object.assign({}, serialized, {
              children: undefined,
              childCount: node.children ? node.children.length : 0,
            });
          }
          if (serialized.children) {
            const childNodes = await Promise.all(
              serialized.children.map((child: any) =>
                figma.getNodeByIdAsync(child.id),
              ),
            );
            const serializedChildren = await Promise.all(
              childNodes
                .filter((n) => n !== null && n.type !== "DOCUMENT")
                .map((n) => serializeWithDepth(n, currentDepth + 1)),
            );
            return Object.assign({}, serialized, { children: serializedChildren });
          }
          return serialized;
        }

        const serialized = await serializeForDetail(node);
        const hasChildren = "children" in node && node.children.length > 0;
        if (!hasChildren) return serialized;
        if (currentDepth >= depth) {
          return Object.assign({}, serialized, { childCount: node.children.length });
        }
        const serializedChildren = await Promise.all(
          node.children
            .filter((n: any) => n.type !== "DOCUMENT")
            .map((n: any) => serializeWithDepth(n, currentDepth + 1)),
        );
        return Object.assign({}, serialized, { children: serializedChildren });
      };

      const targetNodes = await resolveTargetNodes(request);
      const primaryNode = targetNodes[0];
      const rawContextNodes = await Promise.all(
        targetNodes.map((node) => serializeWithDepth(node, 0)),
      );
      const { tree: dedupedNodes, globalVars } = deduplicateStyles({ children: rawContextNodes });
      const contextNodes = (dedupedNodes as any).children;
      const context: any = {
        nodes: contextNodes,
        ...(componentDefs.size > 0 ? { componentDefs: Object.fromEntries(componentDefs) } : {}),
        ...(globalVars ? { globalVars } : {}),
      };
      const requestMetadata = {
        disableCodeConnect: !!(request.params && request.params.disableCodeConnect),
        forceCode: !!(request.params && request.params.forceCode),
        clientFrameworks: request.params && request.params.clientFrameworks ? request.params.clientFrameworks : undefined,
        clientLanguages: request.params && request.params.clientLanguages ? request.params.clientLanguages : undefined,
        excludeScreenshot: !!(request.params && request.params.excludeScreenshot),
      };
      let metadataMessage =
        "Local fallback response from figma-mcp-go. Code Connect and cloud-only enrichments are not available.";
      let degraded = true;
      let code = JSON.stringify(context, null, 2);
      const serializedNodeCount = countSerializedNodes(context.nodes);
      if (code.length > MAX_DESIGN_CONTEXT_CODE_LENGTH) {
        degraded = true;
        metadataMessage +=
          " Context payload is large; prefer get_metadata plus targeted child get_design_context calls for narrower nodes.";
        code = JSON.stringify(
          {
            notice:
              "Context payload truncated in local fallback. Use get_metadata first, then fetch smaller child nodes with get_design_context.",
            nodeId: primaryNode.id,
            nodeName: primaryNode.name,
            nodeCount: serializedNodeCount,
          },
          null,
          2,
        );
      }
      const data: any = {
        fileKey: null,
        nodeId: primaryNode.id,
        name: primaryNode.name,
        code,
        metadata: {
          source: FALLBACK_SOURCE,
          degraded,
          message: metadataMessage,
          request: requestMetadata,
          nodeCount: serializedNodeCount,
          currentPage: {
            id: figma.currentPage.id,
            name: figma.currentPage.name,
          },
        },
        context,
      };
      if (!(request.params && request.params.excludeScreenshot)) {
        data.screenshot = await exportNodeScreenshot(primaryNode);
      }
      return {
        type: request.type,
        requestId: request.requestId,
        data,
      };
    }

    case "get_metadata": {
      const targetNodes = await resolveTargetNodes(request);
      const kind =
        request.params && request.params.nodeId
          ? "node"
          : targetNodes[0]?.type === "PAGE"
            ? "page"
            : "selection";
      return {
        type: request.type,
        requestId: request.requestId,
        data: {
          fileKey: null,
          nodeId: targetNodes[0]?.id ?? null,
          kind,
          source: FALLBACK_SOURCE,
          degraded: true,
          metadataText: buildMetadataText(kind, targetNodes),
          pages: getPageSummaries(),
        },
      };
    }

    case "get_figjam": {
      if (figma.editorType !== "figjam") {
        throw new Error("get_figjam is only available in FigJam files");
      }
      const nodeId =
        request.params && request.params.nodeId && request.params.nodeId !== "0:1"
          ? request.params.nodeId
          : null;
      const targetNodes = nodeId
        ? [await getNodeById(nodeId)]
        : await resolveTargetNodes(request);
      const primaryNode = targetNodes[0];
      const tree = await Promise.all(targetNodes.map((node) => serializeNode(node)));
      const data: any = {
        fileKey: null,
        nodeId: primaryNode.id,
        editorType: figma.editorType,
        source: FALLBACK_SOURCE,
        degraded: true,
        includeImagesOfNodes: !!(request.params && request.params.includeImagesOfNodes),
        metadataText: buildFigJamMetadataText(targetNodes),
        context: tree,
        tree,
      };
      if (request.params && request.params.includeImagesOfNodes) {
        data.images = await Promise.all(targetNodes.map((node) => exportNodeScreenshot(node)));
      }
      return {
        type: request.type,
        requestId: request.requestId,
        data,
      };
    }

    case "get_pages":
      return {
        type: request.type,
        requestId: request.requestId,
        data: {
          currentPageId: figma.currentPage.id,
          pages: figma.root.children.map((page) => ({
            id: page.id,
            name: page.name,
          })),
        },
      };

    case "get_viewport":
      return {
        type: request.type,
        requestId: request.requestId,
        data: {
          center: { x: figma.viewport.center.x, y: figma.viewport.center.y },
          zoom: figma.viewport.zoom,
          bounds: {
            x: figma.viewport.bounds.x,
            y: figma.viewport.bounds.y,
            width: figma.viewport.bounds.width,
            height: figma.viewport.bounds.height,
          },
        },
      };

    case "get_fonts": {
      const fontMap = new Map<string, any>();
      const collectFonts = (n: any) => {
        if (n.type === "TEXT") {
          const fontName = n.fontName;
          if (typeof fontName !== "symbol" && fontName) {
            const key = `${fontName.family}::${fontName.style}`;
            if (!fontMap.has(key)) {
              fontMap.set(key, { family: fontName.family, style: fontName.style, nodeCount: 0 });
            }
            fontMap.get(key).nodeCount++;
          }
        }
        if ("children" in n) n.children.forEach(collectFonts);
      };
      collectFonts(figma.currentPage);
      const fonts = Array.from(fontMap.values()).sort((a, b) => b.nodeCount - a.nodeCount);
      return {
        type: request.type,
        requestId: request.requestId,
        data: { count: fonts.length, fonts },
      };
    }

    case "search_nodes": {
      const query = request.params && request.params.query
        ? request.params.query.toLowerCase()
        : "";
      const scopeNodeId = request.params && request.params.nodeId;
      const types = request.params && request.params.types ? request.params.types : [];
      const limit = request.params && request.params.limit ? request.params.limit : 50;
      const root = scopeNodeId
        ? await figma.getNodeByIdAsync(scopeNodeId)
        : figma.currentPage;
      if (!root) throw new Error(`Node not found: ${scopeNodeId}`);
      const results: any[] = [];
      const search = async (n: any) => {
        if (results.length >= limit) return;
        if (n !== root) {
          const nameMatch = !query || n.name.toLowerCase().includes(query);
          const typeMatch = types.length === 0 || types.includes(n.type);
          if (nameMatch && typeMatch) {
            results.push({
              id: n.id,
              name: n.name,
              type: n.type,
              bounds: getBounds(n),
            });
          }
        }
        if (results.length < limit && "children" in n) {
          for (const child of n.children) await search(child);
        }
      };
      await search(root);
      return {
        type: request.type,
        requestId: request.requestId,
        data: { count: results.length, nodes: results },
      };
    }

    case "get_reactions": {
      const nodeId = request.nodeIds && request.nodeIds[0];
      if (!nodeId) throw new Error("nodeId is required for get_reactions");
      const node = await figma.getNodeByIdAsync(nodeId);
      if (!node || node.type === "DOCUMENT") throw new Error(`Node not found: ${nodeId}`);
      const reactions = "reactions" in node ? node.reactions : [];
      return {
        type: request.type,
        requestId: request.requestId,
        data: { nodeId: node.id, name: node.name, reactions },
      };
    }

    case "scan_text_nodes": {
      const nodeId = request.params && request.params.nodeId;
      if (!nodeId) throw new Error("nodeId is required for scan_text_nodes");
      const root = await figma.getNodeByIdAsync(nodeId);
      if (!root) throw new Error(`Node not found: ${nodeId}`);
      const textNodes: any[] = [];
      const findText = async (n: any) => {
        if (n.type === "TEXT") {
          textNodes.push({
            id: n.id,
            name: n.name,
            characters: n.characters,
            fontSize: isMixed(n.fontSize) ? "mixed" : n.fontSize,
            fontName: isMixed(n.fontName) ? "mixed" : n.fontName,
          });
        }
        if ("children" in n)
          for (const child of n.children) await findText(child);
      };
      figma.ui.postMessage({
        type: "progress_update",
        requestId: request.requestId,
        progress: 10,
        message: "Scanning text nodes...",
      });
      await new Promise((r) => setTimeout(r, 0));
      await findText(root);
      return {
        type: request.type,
        requestId: request.requestId,
        data: { count: textNodes.length, textNodes },
      };
    }

    case "scan_nodes_by_types": {
      const nodeId = request.params && request.params.nodeId;
      const types =
        request.params && request.params.types ? request.params.types : [];
      if (!nodeId)
        throw new Error("nodeId is required for scan_nodes_by_types");
      if (types.length === 0)
        throw new Error("types must be a non-empty array");
      const root = await figma.getNodeByIdAsync(nodeId);
      if (!root) throw new Error(`Node not found: ${nodeId}`);
      const matchingNodes: any[] = [];
      const findByTypes = async (n: any) => {
        if ("visible" in n && !n.visible) return;
        if (types.includes(n.type)) {
          matchingNodes.push({
            id: n.id,
            name: n.name,
            type: n.type,
            bbox: {
              x: "x" in n ? n.x : 0,
              y: "y" in n ? n.y : 0,
              width: "width" in n ? n.width : 0,
              height: "height" in n ? n.height : 0,
            },
          });
        }
        if ("children" in n)
          for (const child of n.children) await findByTypes(child);
      };
      figma.ui.postMessage({
        type: "progress_update",
        requestId: request.requestId,
        progress: 10,
        message: `Scanning for types: ${types.join(", ")}...`,
      });
      await new Promise((r) => setTimeout(r, 0));
      await findByTypes(root);
      return {
        type: request.type,
        requestId: request.requestId,
        data: {
          count: matchingNodes.length,
          matchingNodes,
          searchedTypes: types,
        },
      };
    }

    default:
      return null;
  }
};
