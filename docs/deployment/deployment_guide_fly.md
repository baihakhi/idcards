# Deployment Guide – Fly.io + Cloudflare + R2

This document summarizes the **end‑to‑end deployment steps** for the ID Card application using **Fly.io**, **Cloudflare DNS**, and **Cloudflare R2 + Worker (CDN Edge)**.

---

## 1. Application Hosting (Fly.io)

### 1.1 Build & Deploy App

Your Go application is deployed as a container on Fly.io.

```bash
flyctl launch
flyctl deploy #redeploy
```

This creates:

- A Fly.io app (e.g. `idcard.fly.dev`)
- HTTPS endpoint
- Managed VM + networking

### 1.2 Verify App

```bash
flyctl status
flyctl logs
```

Confirm:

- App is running
- Port 8080 is exposed
- `/` loads correctly

---

## 2. Domain Registration & DNS (Cloudflare)

### 2.1 Domain Setup

- Domain purchased from **Exabytes**
- Nameservers updated to **Cloudflare**

Cloudflare now manages:

- DNS
- TLS
- Proxy / CDN

---

## 3. Connect Domain to Fly.io

### 3.1 Add Domain to Fly.io

```bash
flyctl certs add mydomain.my.id
flyctl certs add www.mydomain.my.id
```

### 3.2 DNS Records in Cloudflare

**A Records** (Proxied ON):

| Type | Name           | Value    |
| ---- | -------------- | -------- |
| A    | mydomain.my.id | Fly IPv4 |
| A    | www            | Fly IPv4 |

**AAAA Record** (optional but recommended):

| Type | Name           | Value    |
| ---- | -------------- | -------- |
| AAAA | mydomain.my.id | Fly IPv6 |

**ACME Validation (auto‑added by Fly):**

```text
_acme-challenge.mydomain.my.id → mydomain.my.id.<hash>.flydns.net
```

### 3.3 Verify TLS

```bash
flyctl certs list
```

Expected:

- Certificate status: **Ready**

---

## 4. Static File Storage (Cloudflare R2)

### 4.1 Create R2 Bucket

- Create bucket (e.g. `idcard-assets`)
- Enable **Custom Domain**

Example:

```
bucket.mydomain.my.id
```

> Note: Custom domain allows public access without exposing R2 endpoint.

---

## 5. Cloudflare Worker (Edge CDN)

### 5.1 Purpose

Worker acts as:

- CDN edge
- Access layer to R2
- Cache controller
- Optional auth / signed URLs

### 5.2 Worker Code

```js
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const objectKey = url.pathname.replace(/^\/static\/uploads\//, "images/");

    const object = await env.MY_BUCKET.get(objectKey);
    if (!object) {
      return new Response("File not found", { status: 404 });
    }

    const headers = new Headers();
    object.writeHttpMetadata(headers);
    headers.set("Access-Control-Allow-Origin", "*");
    headers.set("Cache-Control", "public, max-age=3600");
    headers.set("Content-Disposition", `inline; filename="${objectKey}"`);

    return new Response(object.body, { headers });
  },
};
```

### 5.3 Bindings

Worker → R2 Binding:

```text
MY_BUCKET → idcard-assets
```

---

## 6. Worker Custom Domain (Recommended)

Instead of using root domain, create a **dedicated CDN subdomain**:

```
cdn.mydomain.my.id
```

Cloudflare → Workers → Custom Domains:

- Attach `cdn.mydomain.my.id`

### File Access Pattern

```text
https://cdn.mydomain.my.id/static/uploads/S001.png
```

Benefits:

- No Fly.io traffic for images
- Free R2 egress (inside Cloudflare)
- Aggressive caching
- Clean separation of concerns

---

## 7. Backend Upload Flow (Go App)

### Upload Target

Your Go backend uploads **directly to R2 API**, not via CDN domain:

```text
https://<ACCOUNT_ID>.r2.cloudflarestorage.com/<bucket>/<path>
```

Reason:

- Worker/CDN is **read‑only** for public access
- Backend needs authenticated S3 API

---

## 8. Environment Variables

Set in Fly.io:

```bash
flyctl secrets set \
R2_ACCOUNT_ID=xxx \
R2_ACCESS_KEY=xxx \
R2_SECRET_KEY=xxx \
R2_BUCKET=idcard-assets
```

Use env vars for:

- Paths
- Credentials
- Environment portability

---

## 9. Final Architecture

```
Browser
  ↓
Cloudflare DNS
  ├── mydomain.my.id → Fly.io (Go App)
  └── cdn.mydomain.my.id → Worker → R2
```

---

## 10. Status Checklist

- ✅ Fly.io app running
- ✅ Domain connected via Cloudflare
- ✅ TLS active
- ✅ R2 bucket created
- ✅ Worker deployed
- ✅ CDN subdomain serving images

---

## Notes / Future Enhancements

- Signed URLs (private files)
- Cache versioning
- Image resizing worker
- Access logging
- Rate limiting

---

**Deployment complete 🎉**
