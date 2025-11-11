---
layout: post
title: "RSS 2.0 vs Atom: The Relative URL Problem"
date: 2022-10-12
author: Waken
tags: [rss, atom, web]
comments: true
---

Was reading about RSS vs Atom feed formats. Found out RSS 2.0 has a fundamental problem with relative URLs.

<!-- more -->

## The Problem with RSS 2.0

From the [RSS/Atom comparison](http://www.intertwingly.net/wiki/pie/Rss20AndAtom10Compared):

> RSS 2.0 does not specify the handling of relative URI references. Different feed readers implement differing heuristics for their interpretation. There is no interoperability. In practice, relative URI references cannot be used in RSS feeds.

So if your RSS feed has:
```xml
<link>../images/photo.jpg</link>
```

Some readers might interpret it correctly, others won't. No standard behavior.

## Atom's Solution

Atom 1.0 uses XML's built-in `xml:base` attribute:

```xml
<feed xml:base="https://example.com/blog/">
  <entry>
    <link href="images/photo.jpg" />  <!-- Resolves to https://example.com/blog/images/photo.jpg -->
  </entry>
</feed>
```

Clear specification. All Atom readers handle it the same way.

## Practical Takeaway

For RSS 2.0 feeds: **always use absolute URLs**. Don't rely on relative paths.

```xml
<!-- Bad (RSS 2.0) -->
<link>../images/photo.jpg</link>

<!-- Good (RSS 2.0) -->
<link>https://example.com/images/photo.jpg</link>
```

One of those "hidden gotchas" that explains why some feeds work better than others.
