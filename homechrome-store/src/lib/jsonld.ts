/**
 * Safely serialises a JSON-LD payload for inline use inside a <script> tag.
 *
 * JSON.stringify does NOT escape the sequence "</script>", so user-controlled
 * strings (e.g. product name / description / URL fields) can break out of the
 * script block and execute arbitrary JavaScript.  This helper applies the same
 * Unicode-escape mitigations that Next.js uses for its own inlined JSON.
 */
export function safeJsonLd(data: unknown): string {
  // Use RegExp constructor so that U+2028/U+2029 are expressed as escape
  // sequences in the source code rather than literal characters that the
  // TypeScript parser would treat as line terminators inside a regex literal.
  const LS = new RegExp("\u2028", "g");
  const PS = new RegExp("\u2029", "g");
  return JSON.stringify(data)
    .replace(/</g, "\\u003c")
    .replace(/>/g, "\\u003e")
    .replace(/&/g, "\\u0026")
    .replace(LS, "\\u2028")
    .replace(PS, "\\u2029");
}
