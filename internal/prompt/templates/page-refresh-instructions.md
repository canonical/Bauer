# BAU Page Refresh Implementation Instructions

You are assisting with implementing feedback from a Google Doc into a web project that uses the Vanilla Framework from Canonical. The feedback is provided as structured suggestions in JSON format. 

Your task is to accurately apply these suggestions to the correct files in the project repository. Once you read and understand this document, implement all of the suggestions in the provided JSON data, and follow the instructions carefully.

## Project Context

- **Framework**: Vanilla Framework (https://vanillaframework.io/)
- **Template Engine**: Jinja2
- **Repository**: Current working directory (ensure you're in the target repo)
- **Branch**: Currently active branch (ensure you're on the correct branch)
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
        "type": "insert|delete|replace",
        "original_text": "text to remove/replace",  // Empty for inserts
        "new_text": "text to add/replace with"      // Empty for deletes
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

### Application Process

For each suggestion:

1. **Locate the text**:
   - Search for: `{preceding_text}{original_text}{following_text}`
   - The anchor texts are exact strings from the document

2. **Apply the change** based on type:
   - **insert**: Add `new_text` between `preceding_text` and `following_text`
   - **delete**: Remove `original_text`, keeping anchors intact
   - **replace**: Substitute `original_text` with `new_text`

3. **Verify**:
   - Confirm the resulting text matches `text_after_change`
   - If mismatch, report the discrepancy

### Important Notes

- **Preserve formatting**: Maintain HTML structure, indentation, and styling
- **Exact matching**: Anchor texts are precise - use them to find locations
- **Order matters**: Process suggestions in the order provided
- **Pattern awareness**: If `table_title` indicates a Vanilla pattern, consult the patterns reference below
- **Style changes**: Some suggestions may be style-only changes (e.g., making text bold, adding emphasis). Use appropriate Vanilla Framework classes and HTML to apply these changes.
- **Section deletions**: It is expected that some suggestions involve removing entire sections, this is acceptable behavior, ensure proper HTML structure and semantics are maintained. 

## Vanilla Framework Patterns

When implementing pattern-related changes (identified by `table_title` in location metadata):

1. **Identify the pattern**: Check the `table_title` field (e.g., "Hero", "Equal Heights")
2. **Match with reference**: Find the corresponding pattern in the Vanilla Patterns Reference section that follows these instructions
3. **Apply correctly**: Follow the pattern's structure, required params, and slots
4. **Import macros**: Ensure proper Jinja macro imports at the top of the template

Common patterns you'll encounter:
- **Hero**: Prominent banner with title, description, CTA, images
- **Equal Heights**: Grid of cards/tiles with consistent heights
- **Text Spotlight**: Callout list highlighting benefits (2-7 items)
- **Logo Section**: Partner/client logo displays
- **Tab section**: Navigation or content panes
- **Basic Section**: Flexible 2-column content sections

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

3. **Pattern not recognized**:
   - Check the Vanilla Patterns section below
   - If pattern is missing, implement as generic HTML
   - Flag for review

4. **Verification mismatch**:
   - Report expected vs actual text
   - Indicate which suggestion failed
   - Continue with remaining suggestions

5. **Pattern includes images**:
  - Add a placeholder and use the attached Alt text if found
  - Report this to the user in your summary
  - Continue with the remaining suggestions

## Document Structure

This prompt is organized in the following order:

1. **These instructions** (what you're reading now)
2. **Vanilla Framework Patterns Reference** (reference material for implementing patterns)
3. **Suggestions Data** (JSON array of changes to implement)

## Processing Instructions

**Chunk {{.ChunkNumber}} of {{.TotalChunks}}**

After reviewing the Vanilla Framework Patterns Reference section, process the suggestions data at the end of this document one location at a time. After processing ALL locations in this chunk, report:
- Number of locations processed
- Number of successful changes
- Any errors or issues encountered
- For each chunk, report if a vanilla pattern was changed or added and which one
