Rules acknowledged ✅

# Vanilla patterns

This file summarizes common Vanilla patterns and how to use them from Jinja macros. Each pattern below contains:
- purpose (one line),
- required params / slots,
- minimal Jinja import + usage examples,
- short configuration notes.

Table of contents
- [Hero pattern](#hero-pattern)
- [Equal heights](#equal-heights)
- [Text Spotlight](#text-spotlight)
- [Logo section](#logo-section)
- [Tabs](#tabs)
- [Basic section](#basic-section)

---

## Hero pattern

Purpose: prominent banner (h1) with optional subtitle, description, CTA and image(s).

Key points
- Required param: `title_text` (renders as `h1`).
- Layouts: `'50/50'`, `'50/50-full-width-image'`, `'75/25'`, `'25/75'`, `'fallback'`.
- Flags: `is_split_on_medium` (bool), `display_blank_signpost_image_space` (bool).
- Slots (callable): `description`, `cta`, `image`, `signpost_image`.

Jinja import
```/dev/null/hero-import.jinja#L1-3
{% from "_macros/vf_hero.jinja" import vf_hero %}
```

Minimal usage
```/dev/null/hero-example.jinja#L1-20
{% from "_macros/vf_hero.jinja" import vf_hero %}

{% call(slot) vf_hero(
  title_text='Welcome to our product',
  subtitle_text='Short subtitle',
  layout='50/50',
  is_split_on_medium=true
) %}
  {% call(description) %}<p>Short description.</p>{% endcall %}
  {% call(cta) %}<a class="p-button" href="/signup">Get started</a>{% endcall %}
  {% call(image) %}<img src="/assets/hero.jpg" alt="Hero" />{% endcall %}
{% endcall %}
```

Notes
- For `25/75` provide `signpost_image` or set `display_blank_signpost_image_space=true`.
- For full-width images use `50/50-full-width-image` or place an image container at the same level as the grid columns.
- Import full Vanilla SCSS for consistent styling.

---

## Equal heights

Purpose: grid of item tiles with consistent heights (useful for features, cards).

Key points
- Required params: `title_text`, `items` (Array<Object>).
- Common item fields: `title_text`, `title_link_attrs`, `description_html`, `image_html`, `cta_html`.
- Image aspect controls: `image_aspect_ratio_small`, `image_aspect_ratio_medium`, `image_aspect_ratio_large`.
- Option: `highlight_images` (boolean) to style illustrations.

Jinja import
```/dev/null/equal-heights-import.jinja#L1-3
{% from "_macros/vf_equal-heights.jinja" import vf_equal_heights %}
```

Minimal usage (inline data)
```/dev/null/equal-heights-example.jinja#L1-18
{% from "_macros/vf_equal-heights.jinja" import vf_equal_heights %}

{% set items = [
  {'title_text':'Item 1','image_html':'<img src=\"/img/1.jpg\" alt=\"\">','description_html':'<p>Desc 1</p>'},
  {'title_text':'Item 2','image_html':'<img src=\"/img/2.jpg\" alt=\"\">','description_html':'<p>Desc 2</p>'},
  {'title_text':'Item 3','image_html':'<img src=\"/img/3.jpg\" alt=\"\">','description_html':'<p>Desc 3</p>'}
] %}

{{ vf_equal_heights(title_text='Our features', items=items, highlight_images=False) }}
```

Notes
- Prefer consistent properties across `items` for visual rhythm.
- If number of items is divisible by 4/3, layout adjusts to 4/3 columns on large screens.

---

## Text Spotlight

Purpose: callout list of short items (2–7), used to highlight benefits or actions.

Key points
- Required params: `title_text`, `list_items` (Array<string>).
- Option: `item_heading_level` (2 or 4) — controls item styling.

Jinja import
```/dev/null/text-spotlight-import.jinja#L1-3
{% from "_macros/vf_text-spotlight.jinja" import vf_text_spotlight %}
```

Minimal usage
```/dev/null/text-spotlight-example.jinja#L1-12
{% from "_macros/vf_text-spotlight.jinja" import vf_text_spotlight %}

{{ vf_text_spotlight(
  title_text='Why choose us',
  list_items=['Fast setup','Secure by default','Enterprise support'],
  item_heading_level=2
) }}
```

Notes
- `list_items` may contain HTML strings if needed.
- Use `item_heading_level=4` for more compact item headings.

---

## Logo section

Purpose: heading + optional description + a block of logos or CTA blocks (use for partner/client logos).

Key points
- Required param: `title` (Object with `text` and optional `link_attrs`).
- Required param: `blocks` — Array of block objects (`type: "logo-block"` or `type: "cta-block"`).
- Slot: `description` (optional) for descriptive paragraphs.
- `padding` can be `'default'` or `'deep'`.

Jinja import
```/dev/null/logo-section-import.jinja#L1-3
{% from "_macros/vf_logo-section.jinja" import vf_logo_section %}
```

Minimal usage (blocks array)
```/dev/null/logo-section-example.jinja#L1-22
{% from "_macros/vf_logo-section.jinja" import vf_logo_section %}

{% set blocks = [
  {'type':'logo-block','item':{'logos':[{'attrs':{'src':'/logos/a.svg','alt':'A'}},{'attrs':{'src':'/logos/b.svg','alt':'B'}}]}},
  {'type':'cta-block','item':{'primary':{'content_html':'Contact us','attrs':{'href':'/contact'}}}}
] %}

{{ vf_logo_section(title={'text':'Trusted by'}, blocks=blocks, padding='default') }}
```

Notes
- Use `logo-block` for simple logo lists; `cta-block` to add buttons/links.
- The macro applies section padding automatically; you can override with `padding`.

---

## Tabs

Purpose: navigation or content panes controlled by a tab list.

Key points
- Tabs have two contexts: navigation (links) and content (interactive panes).
- For interactive content panes, JavaScript is required for keyboard navigation and panel toggling.
- Mark the active tab with `aria-selected="true"` on the corresponding `<li>`.

SCSS import
```/dev/null/tabs-scss.jinja#L1-6
// import Vanilla and include base mixins
@import 'vanilla-framework';
@include vf-base;
@include vf-p-tabs;
```

JS import (interactive)
```/dev/null/tabs-js.jinja#L1-3
import { tabs } from 'vanilla-framework/js/tabs';
// Initialize on DOM ready for your tab containers
tabs();
```

Minimal markup (static)
```/dev/null/tabs-markup.html#L1-20
<ul class="p-tabs" role="tablist">
  <li role="tab" aria-selected="true">Tab A</li>
  <li role="tab" aria-selected="false">Tab B</li>
  <li role="tab" aria-selected="false">Tab C</li>
</ul>

<div class="p-tabs__panels">
  <div role="tabpanel">Content A</div>
  <div role="tabpanel" hidden>Content B</div>
  <div role="tabpanel" hidden>Content C</div>
</div>
```

Notes
- For accessible keyboard navigation follow WAI-ARIA tab pattern (arrow keys, home/end).
- Import and initialize the JS module for interactive behaviour.

---

## Basic section

Purpose: flexible 2-column (default) content section composed of various content blocks (description, images, lists, code, logos, CTAs).

Key points
- Required param: `title` (Object with `text` and optional `link_attrs`).
- Common params: `label_text`, `subtitle`, `items` (Array of block objects), `is_split_on_medium`, `padding`, `top_rule_variant`.
- Blocks support `type` keys: `description`, `image`, `video`, `list`, `code-block`, `logo-block`, `cta-block`, `notification`, etc.

Jinja import
```/dev/null/basic-section-import.jinja#L1-3
{% from "_macros/vf_basic-section.jinja" import vf_basic_section %}
```

Minimal usage (mixed items)
```/dev/null/basic-section-example.jinja#L1-28
{% from "_macros/vf_basic-section.jinja" import vf_basic_section %}

{% set items = [
  {'type':'description','item':{'type':'text','content':'Intro paragraph.'}},
  {'type':'image','item':{'attrs':{'src':'/img/feature.jpg','alt':'Feature'},'aspect_ratio':'3-2'}},
  {'type':'cta-block','item':{'primary':{'content_html':'Try demo','attrs':{'href':'/demo'}}}}
] %}

{{ vf_basic_section(title={'text':'Section title'}, items=items, is_split_on_medium=true) }}
```

Notes
- Use `is_split_on_medium=true` to split layout on tablet and larger.
- Items are rendered in sequence; each item supports its own options (see macros for details).
- Import full Vanilla SCSS for images/utility classes used by blocks.

---

General notes
- Always import the appropriate macro from `_macros/*.jinja`.
- Patterns rely on Vanilla CSS utilities — recommended to import the full framework or required partials in your project SCSS.
- When a pattern provides named slots (callable blocks), use `{% call(slotname) %}...{% endcall %}` to inject markup.
- Keep content structure consistent across repeated items to maintain visual rhythm.
