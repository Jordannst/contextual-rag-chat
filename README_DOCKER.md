# Docker Setup untuk Enterprise RAG Chatbot

Panduan untuk menjalankan aplikasi menggunakan Docker Compose.

## Prerequisites

- Docker Desktop (Windows/Mac) atau Docker Engine (Linux)
- Docker Compose v3.8 atau lebih baru

## Setup Awal

1. **Buat file `.env` di root project** (jika belum ada):

```env
# Database
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=rag_chatbot

# Backend
BACKEND_PORT=5000
GIN_MODE=release

# Frontend
FRONTEND_PORT=3000

# API Keys
GEMINI_API_KEY=your_gemini_api_key_here
GEMINI_API_KEYS=key1,key2,key3  # Optional: multiple keys for rotation
COHERE_API_KEY=your_cohere_api_key_here
```

2. **Pastikan file `.env` juga ada di folder `backend/`** dengan konfigurasi yang sama.

## Menjalankan Aplikasi

### Build dan Start Semua Services

```bash
docker-compose up --build
```

### Start di Background (Detached Mode)

```bash
docker-compose up -d --build
```

### Melihat Logs

```bash
# Semua services
docker-compose logs -f

# Service tertentu
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f db
```

### Stop Services

```bash
docker-compose down
```

### Stop dan Hapus Volumes (Hapus Data Database)

```bash
docker-compose down -v
```

## Akses Aplikasi

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:5000
- **PostgreSQL**: localhost:5432

## Struktur Docker

### Services

1. **db** (PostgreSQL + pgvector)
   - Image: `pgvector/pgvector:pg16`
   - Port: 5432
   - Volume: `pgdata` (persistent storage)

2. **backend** (Go + Python)
   - Build dari `./backend/Dockerfile`
   - Port: 5000
   - Dependencies: Python 3, Tesseract OCR, pandas, matplotlib, seaborn
   - Volume: `./backend/uploads` (untuk file upload)

3. **frontend** (Next.js)
   - Build dari `./my-app/Dockerfile`
   - Port: 3000
   - Dependencies: Node.js 18

## Troubleshooting

### Database tidak connect

```bash
# Cek status database
docker-compose ps db

# Cek logs database
docker-compose logs db

# Restart database
docker-compose restart db
```

### Backend error: Python script not found

Pastikan Python scripts di-copy dengan benar:
```bash
docker-compose exec backend ls -la /app/scripts/
```

### Frontend tidak bisa connect ke backend

Pastikan `NEXT_PUBLIC_API_URL` di `docker-compose.yml` sesuai dengan port backend.

### Rebuild setelah perubahan code

```bash
# Rebuild specific service
docker-compose build backend
docker-compose up -d backend

# Rebuild semua
docker-compose up --build
```

## Development Mode

Untuk development, gunakan volume mounting untuk hot-reload:

```yaml
# Tambahkan di docker-compose.yml untuk development
volumes:
  - ./backend:/app
  - ./my-app:/app
```

## Production Deployment

1. Set `GIN_MODE=release` di `.env`
2. Set `NODE_ENV=production` di environment
3. Gunakan reverse proxy (Nginx) untuk production
4. Setup SSL/TLS certificates
5. Configure proper firewall rules

## Notes

- Uploads directory (`./backend/uploads`) di-mount sebagai volume untuk persistensi
- Database data disimpan di Docker volume `pgdata`
- Semua Python dependencies (pandas, matplotlib, seaborn, pytesseract) sudah terinstall di image
- Tesseract OCR sudah terinstall dengan support untuk English dan Indonesian

