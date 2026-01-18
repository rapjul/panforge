---
title: "My Document"
creator: "Johnny Appleseed"
date: 2026-01-01
description: |
  This is a Markdown document used for testing.
from: markdown+hard_line_breaks+lists_without_preceding_blankline-blank_before_blockquote-blank_before_header
output:
  epub:
    to: epub
    output: "My Document.epub"
    standalone: true
    epub-title-page: true
    top-level-division: chapter
    # epub-cover-image: book-cover.jpeg
    # split-level: 2
  html:
    standalone: true
    output: test-output.html
    # include-in-header:
    #   - foo.css
    #   - bar.js
  latex:
    from: gfm+hard_line_breaks
    toc: true
    toc-depth: 3
    output: test.pdf
    # template: letter.tex
    metadata:
      fontsize: 12pt
---
# My Document

My text.

## My Sub-Header

More of my text.
