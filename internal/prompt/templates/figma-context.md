## Design Context

Design information has been extracted from Figma for the suggestions in this chunk.
{{if .Anchors}}
### Referenced design nodes
{{range .Anchors}}
- **{{.NodeName}}** (node: `{{.NodeID}}`)
{{- end}}
{{end}}
{{if .Screenshots}}
### Screenshots

The following screenshots are available locally for the regions related to this chunk:
{{range .Screenshots}}
- `{{.}}`
{{- end}}

Examine them carefully to validate spacing, component usage, and text content before making changes.
{{end}}
{{if .Comments}}
### Designer comments (treat as hard requirements unless they conflict with the Google Doc)
{{range .Comments}}
- **{{.Author}}**: {{.Message}} _(node: `{{.NodeID}}`)_
{{- end}}

The Google Doc is the canonical intent source. Designer comments are requirements within that intent.
{{end}}
### Instructions for design alignment

- Verify your implementation matches the visual design for the suggestion locations in this chunk.
- Do not invent new UI components if the design shows an existing one.
- If the design shows a spacing or typography token, check whether an equivalent exists in the codebase.
