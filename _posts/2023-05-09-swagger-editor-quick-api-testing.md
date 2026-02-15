---
layout: post
title: "Swagger Editor for Quick API Testing"
date: 2023-05-09
author: Waken
tags: [api, swagger, openapi, tools]
comments: true
---

Found Swagger Editor useful for quickly visualizing and testing API specs.

<!-- more -->

## What It Does

Swagger Editor converts OpenAPI YAML definitions into interactive API documentation.

Write your API spec on the left, see live preview on the right.

## Online Version

No installation needed: [https://editor.swagger.io](https://editor.swagger.io)

## Quick Example

Paste an OpenAPI spec:

```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: Success
```

Instantly see:
- Interactive documentation
- Request/response examples
- Try-it-out functionality

## Use Cases

**Testing third-party APIs:**
- Got an OpenAPI spec from a vendor? Load it in Swagger Editor
- See all endpoints and try them out

**Designing new APIs:**
- Write spec first
- See how it looks before implementing

**Documentation:**
- Generate API docs from your YAML

## Local Setup

For offline use:

```bash
docker run -p 8080:8080 swaggerapi/swagger-editor
```

Access at http://localhost:8080

Quick way to work with OpenAPI specs without building custom tooling.
