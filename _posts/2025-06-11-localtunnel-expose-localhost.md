---
layout: post
title: "LocalTunnel: Quick HTTPS Tunnel to Localhost"
date: 2025-06-11
author: Waken
tags: [localtunnel, https, tunneling, development]
comments: true
---

Needed to share my local dev server externally. LocalTunnel made it instant.

<!-- more -->

## What It Does

[LocalTunnel](https://github.com/localtunnel/localtunnel) exposes your localhost to the internet with a public HTTPS URL.

No config, no account needed.

## Install

```bash
npm install -g localtunnel
```

## Usage

Start your local server:
```bash
# Your app running on port 9051
python -m http.server 9051
```

Create tunnel:
```bash
lt --port 9051
```

Get a URL:
```
your url is: https://random-words-123.loca.lt
```

Share that URL. Anyone can access your localhost.

## Custom Subdomain

Want a predictable URL?

```bash
lt --port 9051 --subdomain green-eagles-happen
```

Get:
```
https://green-eagles-happen.loca.lt
```

Same URL every time (if available).

## Use Cases

**Quick demos:**
- Show client your in-progress work
- No deployment needed

**Webhook testing:**
- Test GitHub webhooks locally
- Stripe, Twilio, etc.

**Mobile testing:**
- Access localhost from phone
- Test responsive design on real devices

## Alternatives

**ngrok:** More features, requires account for custom domains
**Cloudflare Tunnel:** More reliable, but more setup
**LocalTunnel:** Dead simple, free, no account

## Security Note

Anyone with the URL can access your server. Don't expose sensitive data.

Use authentication if needed:
```javascript
// Express example
app.use((req, res, next) => {
  const auth = req.headers['authorization'];
  if (auth !== 'Bearer mysecret') {
    return res.status(401).send('Unauthorized');
  }
  next();
});
```

Quick tool for quick sharing.
