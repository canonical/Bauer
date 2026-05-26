## Design Context

Design information has been extracted from Figma for the suggestions in this chunk.
{{- if .Anchors}}

### Referenced design nodes

{{range .Anchors -}}
- **{{.NodeName}}** (node: `{{.NodeID}}`)
{{end -}}
{{end -}}
{{- if .Screenshots}}

### Screenshots

The following screenshots are available locally for the regions related to this chunk:
{{range .Screenshots -}}
- `{{.}}`
{{end}}
Examine them carefully to validate spacing, component usage, and text content before making changes.
{{end -}}
{{- if .Comments}}

### Designer comments (treat as hard requirements unless they conflict with the Google Doc)

{{range .Comments -}}
- **{{.Author}}**: {{.Message}} _(node: `{{.NodeID}}`)_
{{end}}
The Google Doc is the canonical intent source. Designer comments are requirements within that intent.
{{end}}

### Instructions for design alignment

- Verify your implementation matches the visual design for the suggestion locations in this chunk.
- Do not invent new UI components if the design shows an existing one.
- If the design shows a spacing or typography token, check whether an equivalent exists in the codebase.
  {{if .FigmaURL}}

### If you have access to Figma MCP tools (optional)

If your runtime supports the Figma MCP server (e.g. VS Code Copilot Chat, Cursor, or Claude Code),
you may fetch live data directly from the design file to supplement the stored context above:

`{{.FigmaURL}}`

If you choose to use MCP tools:

- Treat Bauer's stored artifacts (screenshots, design node references, and designer comments above)
  as the **ground truth** for this run.
- If the live MCP view conflicts with stored artifacts, **surface the conflict explicitly** rather
  than silently preferring one over the other.
- Do not rely on MCP tools alone — the stored artifacts are the authoritative reference.

If you do not have access to Figma MCP tools, ignore this section entirely.
{{end}}
