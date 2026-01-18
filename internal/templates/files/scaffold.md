---
title: "Untitled Document"
author: "Your Name"

## List of formats to generate. 
## These keys correspond to entries in your .panforge.yaml or the names of Pandoc output formats.
## To see a list of available output formats, run `pandoc --list-output-formats`.
##
## Example:
## outputs:
##   - pdf
##   - html
##
## Default: pdf
##
outputs:
{{- if .Formats }}
{{- range .Formats }}
  - {{ . }}
{{- end }}
{{- else }}
  - pdf
  # - html
  # - docx
{{- end }}
---

<!-- The title is set in the frontmatter and can be changed by editing the title field in the frontmatter. -->
<!-- # Title -->

**By Author**  
**Released on YYYY-MM-DD**

## Introduction

Write your content here...
