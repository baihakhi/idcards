# ID Card Generator (IDCARD)

A **self-hosted internal ID Card management system** built with **Go**, designed to generate **user ID cards (PNG)** and **PDF forms**, manage user data, and serve assets efficiently using **Fly.io**, **Cloudflare**, and **R2 CDN**.

This project is optimized for:
- Internal/company use
- Low-cost hosting
- High performance
- Clean architecture (handler / service / repository)

---

## ✨ Features

- ✅ User CRUD (Create, Read, Update, Delete)
- ✅ Webcam photo capture (browser-based)
- ✅ ID Card generation (PNG)
- ✅ PDF form generation
- ✅ Bulk upsert via XLSX upload
- ✅ Concurrent processing with worker pool
- ✅ SQLite (local) / PostgreSQL (production-ready)
- ✅ Cloudflare R2 for image & PDF storage
- ✅ Edge CDN via Cloudflare Worker
- ✅ Dockerized & Fly.io ready

---

## 🏗️ Architecture Overview

```
Browser
  ↓
Cloudflare DNS
  ├── mydomain.my.id        → Fly.io (Go App)
  └── cdn.mydomain.my.id    → Cloudflare Worker → R2
```

### Backend Layers

```
cmd/                # App entry point
internal/
  ├── handler/      # HTTP handlers
  ├── service/      # Business logic
  ├── repository/   # Database access
  ├── model/        # Domain models
  ├── config/       # DB & env config
  └── util/         # Helpers (image, pdf, etc)
static/             # Frontend assets
templates/          # HTML templates
```

---

## 🧰 Tech Stack

- **Backend:** Go (net/http)
- **Database:** PostgreSQL (prod-ready)
- **Image Processing:** image, draw, freetype
- **PDF:** gofpdf
- **Excel:** excelize
- **Storage:** Cloudflare R2
- **CDN:** Cloudflare Worker
- **Hosting:** Fly.io
- **Container:** Docker (multi-stage build)

---

## 🚀 Getting Started (Local)

### Prerequisites

- Go 1.22+
- SQLite
- GCC (for CGO / sqlite3)

### Run Locally

```bash
go mod tidy
make run #for windows
```
or
```bash
go run ./cmd
```

App runs at:
```
http://localhost:8080
```

---

## 🐳 Docker

### Build & Run

```bash
docker build -t idcard .
docker run -p 8080:8080 idcard
```

Or using Makefile:

```bash
make build
make run
```

---

## ☁️ Deployment (Fly.io)

```bash
flyctl launch
flyctl deploy
```

Set secrets:

```bash
flyctl secrets set \
R2_ACCOUNT_ID=xxx \
R2_ACCESS_KEY=xxx \
R2_SECRET_KEY=xxx \
R2_BUCKET=idcard-assets
```

---

## 🗄️ File Storage (Cloudflare R2)

### Upload Flow

- Backend uploads via **R2 S3 API**
- Public access served via **Cloudflare Worker + CDN**

### Public Access Pattern

```
https://cdn.mydomain.my.id/static/uploads/S001.png
```

---

## 📦 Bulk Upload (XLSX)

- Upload XLSX via UI
- Parsed using `excelize`
- Concurrent upsert with worker pool
- Single DB transaction

Supports:
- Insert new users
- Update existing users

---

## 🧪 Testing

- Repository & Service layers are interface-based

```bash
go test ./internal/...
```

---

## 🔐 Security Notes

- R2 bucket not accessed directly by public
- CDN layer isolates storage
- Ready for signed URLs if needed

---

## 📄 License

Internal use / private repository

---

**Built for internal efficiency, low cost, and long-term maintainability.** 🚀

