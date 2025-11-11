---
layout: post
title: "Emacs Org Mode: Sharp LaTeX Previews with dvisvgm"
date: 2025-01-22
author: Waken
tags: [emacs, org-mode, latex, dvisvgm]
comments: true
---

Switched Emacs Org mode LaTeX previews from ImageMagick to dvisvgm. Much sharper results.

<!-- more -->

## The Problem

Default LaTeX preview in Org mode uses ImageMagick to convert equations to PNG. Results are blurry when scaled.

## Why dvisvgm is Better

- **Vector graphics**: SVG scales perfectly at any zoom level
- **Sharp rendering**: No pixelation
- **Better font handling**: LaTeX fonts render correctly

## Setup

Install dvisvgm:

```bash
# With BasicTeX
tlmgr install dvisvgm

# Or full TeX Live
brew install texlive  # includes dvisvgm
```

Configure in `~/.doom.d/config.el` (or your Emacs config):

```elisp
(after! org
  ;; Use dvisvgm for LaTeX previews
  (setq org-preview-latex-default-process 'dvisvgm)

  (setq org-preview-latex-process-alist
        '((dvisvgm :programs ("latex" "dvisvgm")
                   :description "dvi > svg"
                   :message "You need to install latex and dvisvgm."
                   :use-xcolor t
                   :image-input-type "dvi"
                   :image-output-type "svg"
                   :image-size-adjust (1.0 . 1.0)
                   :latex-compiler ("latex -interaction nonstopmode -output-directory %o %f")
                   :image-converter ("dvisvgm %f -n -b min -c %S -o %O"))))

  ;; Appearance settings
  (setq org-format-latex-header
        "\\documentclass[preview]{standalone}
         \\usepackage{amsmath}
         \\usepackage{xcolor}")

  (setq org-format-latex-options
        (plist-put org-format-latex-options :scale 2.0))
  (setq org-format-latex-options
        (plist-put org-format-latex-options :background "Transparent")))
```

## Usage

1. Clear old previews: `M-x org-clear-latex-preview`
2. Generate new ones: `M-x org-latex-preview`

Equations now render as crisp SVGs.

## Example

Type in Org mode:
```
\[ E = mc^2 \]
```

Hit `C-c C-x C-l` to preview. Zoom in - still sharp!

## Troubleshooting

If previews don't update:
- Clear cache: `M-x org-clear-latex-preview`
- Check `/var/folders/.../` for generated SVGs
- Test manually: `latex test.tex && dvisvgm test.dvi -o test.svg`

Much better than blurry PNGs.
