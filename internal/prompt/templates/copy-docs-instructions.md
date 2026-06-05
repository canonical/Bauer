# BAU Copy Update Implementation Instructions

You are assisting with implementing copy suggestions from a Google Doc into a web project that uses the Vanilla Framework from Canonical. The feedback is provided as structured suggestions in JSON format. You are only required to update text fields. 

Your task is to accurately apply these suggestions to the correct files in the project repository. Once you read and understand this document, implement all of the suggestions in the provided JSON data, and follow the instructions carefully.

## Project Context

- **Framework**: Vanilla Framework (https://vanillaframework.io/)
- **Template Engine**: Jinja2
- **Repository**: Current working directory (ensure you're in the target repo)
- **Branch**: Currently active branch
- **Document**: {{.DocumentTitle}}

## Finding Target Files

The target file path is specified in the metadata as: **{{.SuggestedURL}}**

### Path Resolution Rules

1. **For most pages**: Create/edit a file with the appropriate page name
   - Example: `ubuntu.com/desktop/upcoming-features` → `templates/desktop/upcoming-features.html`

2. **For index pages**: If the URL path matches a folder name
   - Example: `ubuntu.com/desktop` → `templates/desktop/index.html`
   - Create the folder if it doesn't exist

3. **Nested paths**: Create all necessary parent directories
   - Example: `ubuntu.com/engage/resources/guide` → `templates/engage/resources/guide.html`

### File Location Algorithm

```
Given URL: domain.com/path/to/page

1. Remove domain: /path/to/page
2. Check if file exists at: templates/path/to/page.html
3. If not, check: templates/path/to/page/index.html
4. If creating new file:
   - Create directories: templates/path/to/
   - Create file: page.html (or index.html if path ends with existing folder name)
```

## Understanding the Suggestions JSON Schema

Each suggestion in the JSON array is a **LocationGroupedSuggestions** object with this structure:

```json
{
  "location": {
    "section": "Body",              // Section of document (Body, Header, Footer)
    "parent_heading": "Section Name", // Optional: Nearest heading above
    "heading_level": 2,               // Optional: Heading level (1-6)
    "in_table": false,                // Whether suggestion is in a table
    "in_metadata": false,             // True if suggestion comes from the metadata table
    "table": {                        // Optional: Table context if in_table is true
      "table_title": "Pattern Name",  // Pattern name (Hero, Equal Heights, etc.)
      "row_index": 1,
      "column_index": 2,
      "column_header": "Header",
      "row_header": "Row Label"
    }
  },
  "suggestions": [                    // Array of suggestions for this location
    {
      "id": "suggestion-id",
      "anchor": {
        "preceding_text": "exact text before",
        "following_text": "exact text after"
      },
      "change": {
        "type": "insert|delete|replace|link_add|link_change|link_remove",
        "original_text": "text to remove/replace",  // Empty for inserts
        "new_text": "text to add/replace with",      // Empty for deletes
        "link_text": "visible text the link wraps",  // Link changes only
        "old_url": "previous link URL",               // link_change, link_remove
        "new_url": "new link URL"                     // link_add, link_change
      },
      "verification": {
        "text_before_change": "combined before state",
        "text_after_change": "combined after state"
      },
      "position": {
        "start_index": 123,     // Character index in the document before change. Do not use this to locate text, it's for reference only.
        "end_index": 456        // Character index in the document before change. Do not use this to locate text, it's for reference only. 
      },
      "atomic_count": 1                 // Number of atomic operations merged
    }
  ]
}
```

## Applying Changes

Process the suggestions **one location at a time, in order**. For each location:

1. **Read the location context**: Understand where in the document this is
2. **Process each suggestion** in the `suggestions` array sequentially
3. **Apply each change** following the process below
4. **Verify** before moving to the next suggestion

### Metadata Table Suggestions

If `location.in_metadata` is true, the change came from the document metadata table and likely
does not appear verbatim in the target HTML content. Instead, map the change to metadata tags
in the repo that mirror the metadata table entries.

Use this process:
1. **Identify the metadata key**:
   - Prefer `location.table.row_header` (the left column label in the metadata table).
   - Use `location.table.column_header` only if `row_header` is empty.
2. **Find the tag** in the repo:
   - Search for tags or markers that match the metadata key (they mirror each table entry).
   - Use this strict match order:
     1. Exact text match (`Page title` == `Page title`)
     2. Case-insensitive match (`Page title` == `page title`)
     3. Normalized text match (trim + collapse spaces + remove punctuation)
     4. Snake case match (`Page title` -> `page_title`)
   - Stop at the first successful match; do not keep searching looser matches afterward.
   - Look for HTML comments, data attributes, front-matter fields, or template variables.
3. **Apply the change**:
   - Update the tag value to match the suggestion's `new_text`.
   - Do not use anchor text matching for metadata suggestions.
4. **Verify**:
   - Confirm the tag value matches `text_after_change`.

If multiple files contain the same metadata tag, prefer the file matching `{{.SuggestedURL}}`.
If the tag is missing, report it as an error and continue.

### Application Process

For each suggestion:

1. **Locate the text**:
   - Search for: `{preceding_text}{original_text}{following_text}`
   - The anchor texts are exact strings from the document

2. **Apply the change** based on type:
   - **insert**: Add `new_text` between `preceding_text` and `following_text`. If the change also has a `new_url`, wrap the inserted text: `<a href="{new_url}">{new_text}</a>`.
   - **delete**: Remove `original_text`, keeping anchors intact
   - **replace**: Substitute `original_text` with `new_text`. If the change also has a `new_url`, wrap the new text in `<a href="{new_url}">{new_text}</a>` (same as insert).
   - **link_add**: Wrap `link_text` in `<a href="{new_url}">{link_text}</a>`. Preserve any Vanilla link classes/attributes already used on nearby links.
   - **link_change**: Find the existing `<a>` whose visible text is `link_text` and whose href is `old_url`, then update its href to `new_url`. Do not change the link text.
   - **link_remove**: Unwrap the `<a>` around `link_text` (href `old_url`), keeping the visible text in place.

3. **Verify**:
   - Confirm the resulting text matches `text_after_change`
   - If mismatch, report the discrepancy

### Important Notes

- **Preserve formatting**: Maintain HTML structure, indentation, and styling
- **Exact matching**: Anchor texts are precise - use them to find locations
- **Order matters**: Process suggestions in the order provided
- **Pattern awareness**: If `table_title` indicates a Vanilla pattern, consult the patterns reference below
- **Metadata tags**: For `in_metadata` suggestions, update the matching tag in the target repo instead of searching for anchors
- **Other style changes**: Style-only changes such as bold, italic, or emphasis are currently skipped by the extractor and will not appear in the suggestions. (Hyperlink changes are supported — see `link_add`/`link_change`/`link_remove` above.)
- **Section deletions**: It is expected that some suggestions involve removing entire sections, this is acceptable behavior, ensure proper HTML structure and semantics are maintained. 

## Vanilla Framework Patterns

This section is added so you can understand the vanilla patterns you may encounter. Do NOT change any pattern, and report to the user any pattern mismatches.

To identify a pattern from the suggestion use `table_title` in location metadata and match with the corresponding pattern in the Vanilla Patterns Reference section that follows these instructions.

**Note**: The complete Vanilla Framework Patterns Reference appears immediately after these instructions and before the suggestions data.

## Error Handling

If you encounter issues:

1. **File not found**:
   - Check if the path needs index.html instead
   - Verify parent directories exist
   - Report if the URL structure is ambiguous

2. **Anchor text not found**:
   - The file may have been modified since the doc was created
   - Report the missing anchor and suggestion details
   - Ask for manual verification

3. **Verification mismatch**:
   - Report expected vs actual text
   - Indicate which suggestion failed
   - Continue with remaining suggestions

## Document Structure

This prompt is organized in the following order:

1. **These instructions** (what you're reading now)
2. **Vanilla Framework Patterns Reference** (reference material for implementing patterns)
3. **Suggestions Data** (JSON array of changes to implement)

## Processing Instructions

**Chunk {{.ChunkNumber}} of {{.TotalChunks}}**

Process the suggestions data at the end of this document one location at a time. After processing ALL locations in this chunk, report:
- Number of locations processed
- Number of successful changes
- Any errors or issues encountered
