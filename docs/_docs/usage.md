---
title: Usage
permalink: /docs/usage/
description: Jigyll's command-line interface — build, serve, clean, and help.
---

Jigyll's CLI mirrors Jekyll's most-used commands.

```bash
jigyll build       # build the site in the current directory into _site
jigyll serve       # serve at http://localhost:4000; reload on changes
jigyll clean       # remove the generated _site directory
jigyll help        # list commands
jigyll help build  # help for a specific command
```

## `build`

Renders the site to the destination directory (`_site` by default).

| Flag | Purpose |
| --- | --- |
| `--source`, `-s` | Source directory (default: current) |
| `--destination`, `-d` | Output directory |
| `--drafts` | Render posts in `_drafts` |
| `--future` | Render posts with future dates |
| `--unpublished` | Render pages/posts marked `published: false` |
| `--incremental` | Rebuild only changed files |
| `--watch` | Rebuild on file changes |

Set `JEKYLL_ENV=production` to build in production mode.

## `serve`

Builds the site and serves it locally, rebuilding on change.

| Flag | Purpose |
| --- | --- |
| `--host` | Host to bind (default `127.0.0.1`) |
| `--port` | Port (default `4000`) |
| `--open-uri`, `-o` | Open the site in a browser |
| `--watch` | Watch for changes (on by default) |
