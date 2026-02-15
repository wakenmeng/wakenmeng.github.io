---
layout: post
title: "Browser Cache: Why S3 Policy Changes Don't Show Up"
date: 2022-08-17
author: Waken
tags: [debugging, s3, http, caching]
comments: true
---

Spent an hour debugging why my S3 bucket policy changes weren't working. Turns out: browser cache.

<!-- more -->

## The Problem

Changed S3 bucket policy to make a file public. Tested in Chrome. Still getting 403 Forbidden.

```bash
# Changed policy
aws s3api put-bucket-policy --bucket mybucket --policy ...

# Test in browser
https://mybucket.s3.amazonaws.com/file.pdf
# Still 403?? WHY?
```

Tried everything:
- Double-checked policy JSON
- Verified permissions
- Re-uploaded file
- Still 403!

## The Cause

**Browsers cache everything**, including:
- Pages
- Downloads
- HTTP responses
- Error pages (yes, even 403s!)

Even though S3 policy changed, Chrome was showing me the cached 403 response.

## The Solution

Use `wget` or `curl` instead of browsers for testing:

```bash
# Test with curl (no cache)
curl -I https://mybucket.s3.amazonaws.com/file.pdf

# Output:
HTTP/1.1 200 OK  # It works!
```

Policy was fine all along. Browser was lying to me.

## Force Refresh in Browser

If you must use a browser:

```
Cmd+Shift+R (Mac)
Ctrl+Shift+R (Windows/Linux)
```

Hard refresh bypasses cache. But easier to just use curl.

## Lesson

For testing HTTP changes:
- ✅ `curl` or `wget` - fresh request every time
- ❌ Browser - might show cached response

Browsers are great for end users. Terrible for testing infrastructure changes.

Now I know: S3 policy testing = curl, not Chrome.
