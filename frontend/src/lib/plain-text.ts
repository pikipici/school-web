export type TextToken = { kind: "text"; value: string } | { kind: "link"; value: string; href: string };
const candidate = /https?:\/\/[^\s<>]+/gi;
function tokens(value: string): TextToken[] {
  const out: TextToken[] = []; let at = 0;
  const addText = (text: string) => { if (!text) return; const last = out.at(-1); if (last?.kind === "text") last.value += text; else out.push({ kind: "text", value: text }); };
  for (const match of value.matchAll(candidate)) {
    const start = match.index; let raw = match[0]; let suffix = "";
    while (/[.,!?;:)]$/.test(raw)) { suffix = raw.at(-1) + suffix; raw = raw.slice(0, -1); }
    if (start > at) addText(value.slice(at, start));
    try { const url = new URL(raw); if (url.protocol === "http:" || url.protocol === "https:") out.push({ kind: "link", value: raw, href: raw }); else addText(raw); } catch { addText(raw); }
    addText(suffix); at = start + match[0].length;
  }
  if (at < value.length) addText(value.slice(at)); return out;
}
export function toPlainTextModel(value: string): TextToken[][] { return value.replace(/\r\n?/g, "\n").split(/\n[ \t]*\n+/).map(v => v.trim()).filter(Boolean).map(tokens); }
