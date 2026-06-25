#!/usr/bin/env python3
"""Render README.md into a standalone offline HTML document."""

from __future__ import annotations

import argparse
import html
import re
from pathlib import Path


TITLE = "osu-touch README"


def slugify(text: str, seen: dict[str, int]) -> str:
    plain = re.sub(r"`([^`]*)`", r"\1", text)
    plain = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", plain)
    plain = re.sub(r"[*_~]", "", plain).lower()
    slug = re.sub(r"[^a-z0-9\s-]", "", plain)
    slug = re.sub(r"[\s-]+", "-", slug).strip("-") or "section"

    count = seen.get(slug, 0)
    seen[slug] = count + 1
    if count:
        return f"{slug}-{count}"
    return slug


def render_inline(text: str) -> str:
    placeholders: list[str] = []

    def stash(value: str) -> str:
        placeholders.append(value)
        return f"\x00{len(placeholders) - 1}\x00"

    def render_link(match: re.Match[str]) -> str:
        label = match.group(1)
        href = match.group(2)
        attrs = f'href="{html.escape(href, quote=True)}"'
        if not href.startswith("#"):
            attrs += ' target="_blank" rel="noopener noreferrer"'
        return f"<a {attrs}>{label}</a>"

    text = re.sub(
        r"`([^`]*)`",
        lambda match: stash(f"<code>{html.escape(match.group(1))}</code>"),
        text,
    )

    escaped = html.escape(text)
    escaped = re.sub(
        r"\[([^\]]+)\]\(([^)]+)\)",
        render_link,
        escaped,
    )
    escaped = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", escaped)

    for index, value in enumerate(placeholders):
        escaped = escaped.replace(f"\x00{index}\x00", value)
    return escaped


def render_markdown(markdown: str) -> str:
    lines = markdown.splitlines()
    output: list[str] = []
    heading_ids: dict[str, int] = {}
    paragraph: list[str] = []
    in_list = False
    in_code = False
    code_lang = ""
    code_lines: list[str] = []

    def flush_paragraph() -> None:
        nonlocal paragraph
        if paragraph:
            output.append(f"<p>{render_inline(' '.join(paragraph))}</p>")
            paragraph = []

    def close_list() -> None:
        nonlocal in_list
        if in_list:
            output.append("</ul>")
            in_list = False

    for line in lines:
        fence = re.match(r"^```\s*([A-Za-z0-9_-]*)\s*$", line)
        if fence:
            if in_code:
                lang_attr = f' class="language-{html.escape(code_lang, quote=True)}"' if code_lang else ""
                output.append(f"<pre><code{lang_attr}>{html.escape(chr(10).join(code_lines))}</code></pre>")
                code_lines = []
                code_lang = ""
                in_code = False
            else:
                flush_paragraph()
                close_list()
                in_code = True
                code_lang = fence.group(1)
            continue

        if in_code:
            code_lines.append(line)
            continue

        if not line.strip():
            flush_paragraph()
            close_list()
            continue

        heading = re.match(r"^(#{1,6})\s+(.+?)\s*$", line)
        if heading:
            flush_paragraph()
            close_list()
            level = len(heading.group(1))
            title = heading.group(2)
            heading_id = slugify(title, heading_ids)
            output.append(
                f'<h{level} id="{heading_id}"><a class="heading-link" '
                f'href="#{heading_id}">{render_inline(title)}</a></h{level}>'
            )
            continue

        item = re.match(r"^-\s+(.+)$", line)
        if item:
            flush_paragraph()
            if not in_list:
                output.append("<ul>")
                in_list = True
            output.append(f"<li>{render_inline(item.group(1))}</li>")
            continue

        close_list()
        paragraph.append(line.strip())

    flush_paragraph()
    close_list()
    if in_code:
        lang_attr = f' class="language-{html.escape(code_lang, quote=True)}"' if code_lang else ""
        output.append(f"<pre><code{lang_attr}>{html.escape(chr(10).join(code_lines))}</code></pre>")

    return "\n".join(output)


def render_page(body: str) -> str:
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{html.escape(TITLE)}</title>
  <style>
    :root {{
      color-scheme: light;
      --paper: #fffaf0;
      --ink: #19140e;
      --muted: #695d4b;
      --accent: #d85d27;
      --accent-dark: #8f321a;
      --panel: #fff2d8;
      --panel-strong: #f7d89b;
      --border: #e4bf7d;
      --code-bg: #221a14;
      --code-ink: #ffe9bf;
    }}

    * {{ box-sizing: border-box; }}
    html {{ scroll-behavior: smooth; }}
    body {{
      margin: 0;
      color: var(--ink);
      font: 17px/1.65 Georgia, "Times New Roman", serif;
      background:
        radial-gradient(circle at 12% 8%, rgba(216, 93, 39, 0.18), transparent 28rem),
        radial-gradient(circle at 92% 0%, rgba(247, 216, 155, 0.9), transparent 24rem),
        linear-gradient(135deg, #fffaf0 0%, #ffe9c6 100%);
    }}

    main {{
      width: min(940px, calc(100% - 32px));
      margin: 40px auto;
      padding: clamp(24px, 5vw, 56px);
      background: rgba(255, 250, 240, 0.92);
      border: 1px solid rgba(228, 191, 125, 0.8);
      border-radius: 28px;
      box-shadow: 0 24px 80px rgba(84, 48, 19, 0.16);
    }}

    h1, h2, h3 {{
      font-family: Georgia, "Times New Roman", serif;
      line-height: 1.15;
      margin: 0.85em 0 0.4em;
      letter-spacing: -0.025em;
    }}

    h1 {{
      margin-top: 0;
      font-size: clamp(2.25rem, 7vw, 4.6rem);
      color: var(--accent-dark);
    }}

    h2 {{
      padding-top: 0.42em;
      border-top: 2px solid var(--border);
      font-size: clamp(1.55rem, 3.2vw, 2.25rem);
    }}

    h3 {{
      margin: 0.65em 0 0.42em;
      font-size: clamp(1.08rem, 1.8vw, 1.28rem);
    }}
    p {{ margin: 0 0 0.95em; }}
    ul {{ margin: 0 0 1em; padding-left: 1.55em; }}
    li {{ margin: 0.26em 0; padding-left: 0.38em; }}
    a {{ color: var(--accent-dark); text-decoration-color: rgba(216, 93, 39, 0.38); }}
    a:hover {{ color: var(--accent); }}

    .heading-link {{
      color: inherit;
      text-decoration: none;
      text-decoration-thickness: 0.08em;
      text-underline-offset: 0.12em;
      transition: color 140ms ease, text-decoration-color 140ms ease;
    }}

    .heading-link:hover, .heading-link:focus-visible {{
      color: var(--accent-dark);
      text-decoration: underline;
      text-decoration-color: rgba(216, 93, 39, 0.45);
    }}

    code {{
      padding: 0.12em 0.34em;
      border-radius: 0.35em;
      background: var(--panel);
      font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
      font-size: 0.9em;
    }}

    pre {{
      overflow-x: auto;
      margin: 1em 0 1.25em;
      padding: 1rem;
      border-radius: 18px;
      background: var(--code-bg);
      box-shadow: inset 0 0 0 1px rgba(255, 233, 191, 0.12);
    }}

    pre code {{
      display: block;
      padding: 0;
      background: transparent;
      color: var(--code-ink);
      font-size: 0.88rem;
      line-height: 1.55;
    }}

    body > main > p:first-of-type {{
      font-size: 1.22rem;
      color: #4c3f30;
    }}

    .to-top {{
      position: fixed;
      right: clamp(16px, 3vw, 32px);
      bottom: clamp(16px, 3vw, 32px);
      z-index: 10;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 3rem;
      height: 3rem;
      padding: 0;
      border: 1px solid rgba(255, 233, 191, 0.28);
      border-radius: 999px;
      background: rgba(34, 26, 20, 0.9);
      color: var(--code-ink);
      box-shadow: 0 12px 34px rgba(84, 48, 19, 0.24);
      font: 700 1.35rem/1 Georgia, "Times New Roman", serif;
      text-decoration: none;
      cursor: pointer;
      opacity: 0;
      pointer-events: none;
      transform: translateY(8px);
      transition: opacity 140ms ease, transform 140ms ease, background 140ms ease;
      appearance: none;
    }}

    .to-top.is-visible {{
      opacity: 1;
      pointer-events: auto;
      transform: translateY(0);
    }}

    .to-top:hover, .to-top:focus-visible {{
      background: var(--accent-dark);
      color: #fff7e8;
      transform: translateY(-2px);
    }}

    @media (max-width: 640px) {{
      main {{ width: min(100% - 18px, 940px); margin: 9px auto; border-radius: 20px; }}
      .to-top {{ width: 2.7rem; height: 2.7rem; }}
    }}
  </style>
</head>
<body>
  <main>
{body}
  </main>
  <button class="to-top" type="button" aria-label="Scroll to top">&#8593;</button>
  <script>
    (() => {{
      const button = document.querySelector('.to-top');
      const toggleButton = () => {{
        button.classList.toggle('is-visible', window.scrollY > 240);
      }};

      button.addEventListener('click', () => {{
        window.scrollTo({{ top: 0, behavior: 'smooth' }});
      }});
      window.addEventListener('scroll', toggleButton, {{ passive: true }});
      toggleButton();
    }})();
  </script>
</body>
</html>
"""


def main() -> None:
    parser = argparse.ArgumentParser(description="Render README.md into standalone HTML.")
    parser.add_argument("source", nargs="?", default="README.md", help="Markdown source file")
    parser.add_argument("output", nargs="?", default="README.html", help="HTML output file")
    args = parser.parse_args()

    source = Path(args.source)
    output = Path(args.output)
    body = render_markdown(source.read_text(encoding="utf-8"))
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(render_page(body), encoding="utf-8")


if __name__ == "__main__":
    main()
