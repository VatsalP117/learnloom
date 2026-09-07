// Satoshi (Fontshare) is the dashboard product face. The stylesheet is
// injected only when the authenticated app graph mounts, so signed-out pages
// (/sign-in, /sign-up) never fetch it. The doc parameter keeps the helper
// callable from node-side renders (tests) where no DOM exists.
const SATOSHI_STYLESHEET = "https://api.fontshare.com/v2/css?f[]=satoshi@400,500,700&display=swap";
const STYLESHEET_MARKER = "data-product-stylesheet";

export function loadProductStylesheet(doc: Document | undefined): boolean {
  if (typeof doc === "undefined") return false;
  if (doc.querySelector(`link[${STYLESHEET_MARKER}]`)) return false;
  const link = doc.createElement("link");
  link.rel = "stylesheet";
  link.href = SATOSHI_STYLESHEET;
  link.setAttribute(STYLESHEET_MARKER, "");
  doc.head.append(link);
  return true;
}