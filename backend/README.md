# Backend API - Go

Backend API untuk AI RAG Chatbot menggunakan Go dan Gin framework.

## Requirements

- Go 1.21 atau lebih baru
- PostgreSQL dengan pgvector extension

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Setup environment variables (buat file `.env`):
```
DATABASE_URL=postgresql://user:password@localhost:5432/rag_chatbot
PORT=5000
```

3. Buat database (jika belum ada):
```powershell
# Windows PowerShell
.\scripts.ps1 create-db
# atau langsung
go run cmd/create-db/main.go
```

4. Jalankan migration:
```powershell
# Windows PowerShell
.\scripts.ps1 migrate
# atau langsung
go run cmd/migrate/main.go
```

## Menjalankan Server

Development mode:
```powershell
# Windows PowerShell
.\scripts.ps1 dev
# atau langsung
go run main.go
```

Build dan run:
```powershell
# Windows PowerShell
.\scripts.ps1 build
.\scripts.ps1 run
# atau langsung
go build -o bin/backend.exe main.go
.\bin\backend.exe
```

## API Endpoints

### GET /ping
Health check endpoint.
```json
{
  "message": "pong"
}
```

### POST /api/upload
Upload file (PDF atau TXT) dan extract text.

**Request:**
- Method: POST
- Content-Type: multipart/form-data
- Field: `document` (file)

**Response:**
```json
{
  "fileName": "document-1234567890.pdf",
  "filePath": "uploads/document-1234567890.pdf",
  "text": "Extracted text content..."
}
```

## Struktur Project

```
backend/
├── cmd/
│   ├── create-db/     # Script untuk membuat database
│   └── migrate/       # Script untuk menjalankan migration
├── db/
│   ├── db.go          # Database connection
│   └── migration.sql  # SQL migration file
├── handlers/
│   └── upload.go      # Upload file handler
├── routes/
│   └── routes.go      # Route definitions
├── utils/
│   └── document_extractor.go  # Text extraction utilities
├── main.go            # Entry point
├── go.mod             # Go modules
└── scripts.ps1      # PowerShell scripts (Windows)
```

## Catatan Windows

Jika `make` tidak tersedia, gunakan `scripts.ps1` atau jalankan perintah Go langsung:
- `go run main.go` - Development
- `go build -o bin/backend.exe main.go` - Build
- `go run cmd/create-db/main.go` - Create database
- `go run cmd/migrate/main.go` - Run migration
