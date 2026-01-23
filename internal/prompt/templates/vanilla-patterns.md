# Vanilla patterns

This file summarizes common Vanilla patterns and how to use them from Jinja macros. Each pattern below contains:
- purpose (one line),
- required params / slots,
- minimal Jinja import + usage examples,
- short configuration notes.

You should import all required macros at the beginning of the Jinja template before using them.

Table of contents
- [Hero pattern](#hero-pattern)
- [Equal heights](#equal-heights)
- [Text Spotlight](#text-spotlight)
- [Logo section](#logo-section)
- [Tab section](#tab-section)
- [Tiered list](#tiered-list)
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

Minimal usage (using call syntax with slots)
```/dev/null/equal-heights-example.jinja#L1-40
{% from "_macros/vf_equal-heights.jinja" import vf_equal_heights %}

{% call(slot) vf_equal_heights(
  title_text="Keep this heading to 2 lines on large screens.",
  attrs={ "id": "4-columns-responsive" },
  subtitle_text="Ensure the right hand side of this 50/50 split is taller than the left hand side (heading) on its left. This includes the subtitle and description.",
  items=[
    {
      "title_text": "A strong hardware ecosystem",
      "image_html":  "<img src='https://assets.ubuntu.com/v1/ff6a068d-kernelt-vanilla-ehp-1.png' class='p-image-container__image' width='284' height='426' alt='Kernelt' />",
      "description_html": "<p>We enable Ubuntu Core with the best ODMs and silicon vendors in the world. We continuously test it on leading IoT and edge devices and hardware.</p>",
      "cta_html": "<a href='#'>Browse all certified hardware&nbsp;&rsaquo;</a>"
    },
    {
      "title_text": "A strong hardware ecosystem",
      "image_html":  "<img src='https://assets.ubuntu.com/v1/7aa4ed28-kernelt-vanilla-ehp-2.png' class='p-image-container__image' width='284' height='426' alt='Kernelt' />",
      "description_html": "<p>We enable Ubuntu Core with the best ODMs and silicon vendors in the world. We continuously test it on leading IoT and edge devices and hardware.</p>",
      "cta_html": "<a href='#'>Browse all certified hardware&nbsp;&rsaquo;</a>"
    },
    {
      "title_text": "A strong hardware ecosystem",
      "image_html":  "<img src='https://assets.ubuntu.com/v1/4936d43a-kernelt-vanilla-ehp-3.png' class='p-image-container__image' width='284' height='426' alt='Kernelt' />",
      "description_html": "<p>We enable Ubuntu Core with the best ODMs and silicon vendors in the world. We continuously test it on leading IoT and edge devices and hardware.</p>",
      "cta_html": "<a href='#'>Browse all certified hardware&nbsp;&rsaquo;</a>"
    },
    {
      "title_text": "A strong hardware ecosystem",
      "image_html":  "<img src='https://assets.ubuntu.com/v1/bbe7b062-kernelt-vanilla-ehp-4.png' class='p-image-container__image' width='284' height='426' alt='Kernelt' />",
      "description_html": "<p>We enable Ubuntu Core with the best ODMs and silicon vendors in the world. We continuously test it on leading IoT and edge devices and hardware.</p>",
      "cta_html": "<a href='#'>Browse all certified hardware&nbsp;&rsaquo;</a>"
    }
  ]
) %}
{% endcall %}
```

Notes
- Prefer consistent properties across `items` for visual rhythm.
- If number of items is divisible by 4/3, layout adjusts to 4/3 columns on large screens.
- For the parent `title_text` use the text from the first suggestion in the given location, if it is much shorter than the others.

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

## Logo section (aka logo cloud)

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

Minimal usage (using call syntax with slots)
```/dev/null/logo-section-example.jinja#L1-60
{% from "_macros/vf_logo-section.jinja" import vf_logo_section %}

{% call(slot) vf_logo_section(
  title={
    "text": "The quick brown fox jumps over the lazy dog"
  },
  blocks=[
    {
      "type": "cta-block",
      "item": {
        "primary": {
          "content_html": "Primary Button",
          "attrs": {
            "href": "#"
          }
        },
        "link": {
          "content_html": "Lorem ipsum dolor sit amet ›",
          "attrs": {
            "href": "#"
          }
        }
      }
    },
    {
      "type": "logo-block",
      "item": {
        "logos": [
          {
            "src": "https://assets.ubuntu.com/v1/38fdfd23-Dell-logo.png",
            "alt": "Dell Technologies"
          },
          {
            "src": "https://assets.ubuntu.com/v1/cd5f636a-hp-logo.png",
            "alt": "Hewlett Packard"
          },
          {
            "src": "https://assets.ubuntu.com/v1/f90702cd-lenovo-logo.png",
            "alt": "Lenovo"
          },
          {
            "src": "https://assets.ubuntu.com/v1/2ef3c028-amazon-web-services-logo.png",
            "alt": "Amazon Web Services"
          },
          {
            "src": "https://assets.ubuntu.com/v1/cb7ef8ac-ibm-cloud-logo.png",
            "alt": "IBM Cloud"
          },
          {
            "src": "https://assets.ubuntu.com/v1/210f44e4-microsoft-azure-new-logo.png",
            "alt": "Microsoft Azure"
          },
          {
            "src": "https://assets.ubuntu.com/v1/a554a818-google-cloud-logo.png",
            "alt": "Google Cloud Platform"
          },
          {
            "src": "https://assets.ubuntu.com/v1/b3e692f4-oracle-new-logo.png",
            "alt": "Oracle"
          }
        ]
      }
    }
  ]
) -%}
{%- if slot == 'description' -%}
<p>The quick brown fox jumps over the lazy dog</p>
{%- endif -%}
{% endcall -%}
```

Notes
- Use `logo-block` for simple logo lists; `cta-block` to add buttons/links.
- The macro applies section padding automatically; you can override with `padding`.

---

## Tab section

Purpose: organize related content into separate tabs within a section with title, optional description, and CTA.

Key points
- Required params: `title` (Object with `text`), `tabs` (Array of tab objects).
- Layouts: `'full-width'`, `'50-50'` (default), `'25-75'`.
- Each tab has `type` (content block type) and `item` (block config).
- Supported block types vary by layout (e.g., quote only in full-width).
- JavaScript required for tab interactivity.

Jinja import
```/dev/null/tab-section-import.jinja#L1-3
{% from "_macros/vf_tab-section.jinja" import vf_tab_section %}
```

Minimal usage
```/dev/null/tab-section-example.jinja#L1-40
{% from "_macros/vf_tab-section.jinja" import vf_tab_section %}

{{ vf_tab_section(
  title={"text": "Features"},
  description={"content": "Explore our key features", "type": "text"},
  layout="50-50",
  tabs=[
    {
      "tab_html": "Logos",
      "type": "logo-block",
      "item": {
        "logos": [
          {"attrs": {"src": "https://assets.ubuntu.com/v1/cd5f636a-hp-logo.png", "alt": "HP"}},
          {"attrs": {"src": "https://assets.ubuntu.com/v1/f90702cd-lenovo-logo.png", "alt": "Lenovo"}}
        ]
      }
    },
    {
      "tab_html": "Blog",
      "type": "blog",
      "item": {
        "articles": [
          {
            "title": {"text": "Getting started", "link_attrs": {"href": "#"}},
            "description": {"text": "Learn the basics"},
            "metadata": {
              "authors": [{"text": "Author Name", "link_attrs": {"href": "#"}}],
              "date": {"text": "15 March 2025"}
            }
          }
        ]
      }
    }
  ]
) }}
```

Notes
- Block types: `quote`, `linked-logo`, `logo-block`, `divided-section`, `blog`, `basic-section`.
- Full-width supports quote; 50/50 supports divided-section and basic-section; all support linked-logo, logo-block, blog.
- Requires JS module: `import {tabs} from 'vanilla-framework/js'; tabs.initTabs('[role="tablist"]');`

---

## Tiered list

Purpose: list of paired titles and descriptions with optional top-level description and CTAs.

Key points
- Required params: `is_description_full_width_on_desktop` (bool), `is_list_full_width_on_tablet` (bool).
- Uses slots: `title`, `description` (optional), `list_item_title_[1-25]`, `list_item_description_[1-25]`, `cta` (optional).
- Layouts determined by boolean flags (50/50 vs full-width on different breakpoints).
- Max 25 list items.

Jinja import
```/dev/null/tiered-list-import.jinja#L1-3
{% from "_macros/vf_tiered-list.jinja" import vf_tiered_list %}
```

Minimal usage
```/dev/null/tiered-list-example.jinja#L1-30
{% from "_macros/vf_tiered-list.jinja" import vf_tiered_list %}

{% call(slot) vf_tiered_list(
  is_description_full_width_on_desktop=true,
  is_list_full_width_on_tablet=false
) %}
  {% if slot == 'title' %}
    <h2>Key benefits</h2>
  {% elif slot == 'description' %}
    <p>Discover what makes our solution unique.</p>
  {% elif slot == 'list_item_title_1' %}
    <h3>Fast deployment</h3>
  {% elif slot == 'list_item_description_1' %}
    <p>Get up and running in minutes with our streamlined setup.</p>
  {% elif slot == 'list_item_title_2' %}
    <h3>Enterprise support</h3>
  {% elif slot == 'list_item_description_2' %}
    <p>24/7 support from our expert team.</p>
  {% elif slot == 'cta' %}
    <a href="#" class="p-button--positive">Get started</a>
  {% endif %}
{% endcall %}
```

Notes
- `is_description_full_width_on_desktop=true` makes title/description span full width on desktop.
- `is_list_full_width_on_tablet=false` makes list items display side-by-side on tablet.
- Use numbered slots (`list_item_title_1`, `list_item_description_1`, etc.) for each list item pair.

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
