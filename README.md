# Enterprise RAG Chatbot

A production-grade Retrieval-Augmented Generation (RAG) system that combines semantic document search with intelligent code execution for data analysis. The system provides dual processing pipelines: a RAG pipeline for text documents (PDF, TXT, DOCX) using hybrid vector and full-text search, and a Data Analyst Agent pipeline for structured data (CSV, Excel) that dynamically generates and executes Python code for analytical queries. Built with Go, Next.js, PostgreSQL with pgvector, and Google Gemini AI.

---

## Table of Contents

- [System Architecture](#system-architecture)
- [Key Features](#key-features)
- [Prerequisites](#prerequisites)
- [Installation & Deployment](#installation--deployment)
- [Configuration](#configuration)
- [Usage Guide](#usage-guide)
- [API Documentation](#api-documentation)
- [Project Structure](#project-structure)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

---

## System Architecture

### Hybrid Backend Architecture

The system employs a hybrid architecture combining Go and Python:

- **Go Backend (Server)**: Handles HTTP requests, database operations, AI API interactions, and orchestrates Python script execution
- **Python Scripts (Workers)**: Specialized processing modules for PDF/OCR extraction, data analysis, and code interpretation
- **Communication**: Go backend executes Python scripts via subprocess calls with environment variable passing

### Data Flow: Document Upload

```
User Upload
    ↓
File Type Detection (Go)
    ↓
┌─────────────────────────────────────┐
│  Text Documents (PDF/TXT/DOCX)      │
│  ┌───────────────────────────────┐   │
│  │ Extract Text (Go/Python)      │   │
│  │ → OCR if needed (Tesseract)   │   │
│  │ → Image Description (Gemini)  │   │
│  └───────────────────────────────┘   │
│           ↓                           │
│  Split into Chunks (1000 chars)      │
│  Overlap: 200 chars                  │
│           ↓                           │
│  Generate Embeddings (Gemini)        │
│  768-dimensional vectors              │
│           ↓                           │
│  Store in PostgreSQL + pgvector      │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│  Structured Data (CSV/Excel)         │
│  ┌───────────────────────────────┐   │
│  │ Process with Pandas (Python)  │   │
│  │ → Generate Preview             │   │
│  │ → Store metadata               │   │
│  └───────────────────────────────┘   │
│  (No embedding, processed on-demand) │
└─────────────────────────────────────┘
```

### Data Flow: Chat Query Processing

#### RAG Flow (Text Documents)

```
User Query
    ↓
Query Rewriting (if history exists)
    ↓
Generate Query Embedding (Gemini text-embedding-004)
    ↓
Hybrid Search
    ├─ Vector Search (pgvector cosine distance)
    └─ Full-Text Search (PostgreSQL tsvector)
    ↓
Combined Scoring & Ranking
    ↓
Similarity Threshold Filter (0.65)
    ↓
Optional: Cohere Re-ranking (top 5)
    ↓
Build Context from Top-K Chunks
    ↓
Generate Response (Gemini 2.0 Flash, streaming)
    ↓
Stream to Frontend (SSE)
```

#### Data Analyst Flow (CSV/Excel)

```
User Query + CSV/Excel File
    ↓
Generate File Preview (columns + sample data)
    ↓
AI Code Generation (Gemini 2.0 Flash)
    ↓
Code Validation (security check)
    ↓
Execute Python Code (pandas/numpy/matplotlib/seaborn)
    ↓
Get Python Output (text + chart data)
    ↓
AI Interpretation (convert technical → natural language)
    ↓
Stream Interpreted Response (SSE)
```

---

## Key Features

### Hybrid Search Engine

Combines vector similarity and full-text search with weighted scoring:

- **Vector Search**: Semantic similarity using pgvector cosine distance (70% weight)
- **Full-Text Search**: Keyword matching using PostgreSQL tsvector with GIN indexing (30% weight)
- **Combined Scoring**: Normalized weighted combination: `(1 - vector_distance/2) * 0.7 + text_rank * 0.3`
- **Re-ranking**: Optional Cohere rerank-multilingual-v3.0 for final result optimization
- **Fallback Strategy**: Automatic fallback to vector-only search when hybrid search yields no results

**Implementation**: `backend/db/db.go::SearchHybridDocuments()`

### Multimodal Ingestion Pipeline

Hybrid extraction pipeline supporting both native text and scanned documents:

- **Native Text Extraction**: Direct text extraction from PDF/TXT/DOCX using Go libraries
- **OCR Processing**: Tesseract OCR integration for scanned PDFs with advanced preprocessing:
  - Matrix scaling (3x) for resolution enhancement (72 DPI → 216 DPI)
  - Grayscale conversion for noise reduction
  - Binarization with threshold 150 for text sharpening
  - PSM 6 configuration for tabular document reading
  - Multi-language support (English + Indonesian)
- **Image Analysis**: Gemini Vision API for image description within PDFs
- **Python Integration**: Subprocess execution of Python scripts for specialized processing

**Implementation**: `backend/scripts/pdf_processor.py`, `backend/utils/document_extractor.go`

### Data Analyst Agent

Dynamic Python code execution system for analytical queries on structured data:

- **AI Code Generation**: Natural language to Python code conversion using Gemini 2.0 Flash
- **Code Sanitization**: Security validation blocking dangerous operations (file I/O, system commands, restricted imports)
- **Execution Engine**: Sandboxed Python environment with pandas, numpy, matplotlib, and seaborn access
- **Chart Visualization**: Automatic chart generation with Base64 encoding for frontend display
- **AI Interpretation**: Technical output conversion to natural language for user-friendly responses
- **File Preview Generation**: Automatic structure analysis (columns, sample data) for context

**Flow**: Query → Preview → Generate Code → Validate → Execute → Interpret → Stream

**Implementation**: `backend/scripts/code_interpreter.py`, `backend/utils/code_runner.go`, `backend/utils/ai.go::GenerateAnalysisCode()`

### Resilience Architecture

Production-grade reliability mechanisms:

- **API Key Rotation**: Automatic key rotation on rate limit errors with multiple key support
- **Model Fallback Chain**: Sequential model fallback (gemini-2.0-flash → gemini-2.0-flash-001 → gemini-flash-latest → gemini-2.5-flash)
- **Error Recovery**: Graceful degradation with fallback strategies
- **Key Management**: Singleton KeyManager with thread-safe rotation

**Implementation**: `backend/utils/key_manager.go`

---

## Prerequisites

> [!IMPORTANT]
> Docker Desktop (Windows/Mac) or Docker Engine (Linux) is required for the recommended deployment method. All dependencies (Node.js, Go, Python, PostgreSQL, Tesseract OCR) are included in Docker images.

### Required API Keys

> [!IMPORTANT]
> You must obtain the following API keys before deployment:
> - **Google Gemini API Key**: Required for embeddings, text generation, and code generation. [Get one here](https://makersuite.google.com/app/apikey)
> - **Cohere API Key**: Optional, but recommended for document reranking functionality

### Manual Installation Prerequisites

If deploying without Docker, you will need:

- **Node.js** 20+ and npm
- **Go** 1.24+ ([Download](https://go.dev/dl/))
- **PostgreSQL** 16+ with pgvector extension
- **Python** 3.11+ with pip
- **Tesseract OCR** (for OCR functionality)

> [!NOTE]
> For Windows users performing manual installation, Tesseract OCR must be installed separately. Download the installer from [GitHub](https://github.com/UB-Mannheim/tesseract/wiki) and ensure it is added to your system PATH.

---

## Installation & Deployment

### Docker Compose (Recommended)

The fastest and most reliable deployment method. All services are containerized and pre-configured.

#### Quick Start

1. **Clone Repository**

```bash
git clone <repository-url>
cd ai-rag-chatbot/my-app
```

2. **Create `.env` file** in project root:

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

> [!IMPORTANT]
> The `.env` file **MUST** be located in the **project root** (same level as `docker-compose.yml`), **NOT** in the `backend/` folder. Docker Compose reads environment variables from the `.env` file in the root folder when running `docker-compose up`. If the `.env` file is located in the `backend/` folder, Docker will not be able to read the environment variables.

3. **Build and Start All Services**

```bash
docker-compose up --build
```

> [!TIP]
> Use the `--build` flag to rebuild images when Dockerfiles or dependencies change. For subsequent starts without changes, you can use `docker-compose up` without the flag.

This will automatically:
- Build backend image (Go + Python + Tesseract OCR)
- Build frontend image (Next.js)
- Start PostgreSQL with pgvector extension
- Configure all dependencies and connections

4. **Access Application**

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:5000
- **PostgreSQL**: localhost:5432

#### Docker Commands

```bash
# Start in background (detached mode)
docker-compose up -d --build

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f backend
docker-compose logs -f frontend

# Stop services
docker-compose down

# Stop and remove volumes (delete database)
docker-compose down -v

# Rebuild specific service
docker-compose build --no-cache backend
docker-compose up -d backend
```

### Manual Installation

For development or custom configuration:

#### 1. Clone Repository

```bash
git clone <repository-url>
cd ai-rag-chatbot/my-app
```

#### 2. Backend Setup

```bash
cd backend

# Install Go dependencies
go mod download

# Build application
go build -o backend.exe main.go
```

#### 3. Frontend Setup

```bash
# From project root
npm install
```

#### 4. Python Dependencies

```bash
# Install required Python packages
pip install pandas openpyxl pymupdf pytesseract pillow google-generativeai matplotlib seaborn numpy
```

#### 5. Environment Configuration

Create `.env` file in `backend/` directory:

```env
# Database Configuration
DATABASE_URL=postgresql://postgres:your_password@localhost:5433/rag_chatbot

# Google Gemini API
# Option 1: Single key
GEMINI_API_KEY=your_gemini_api_key_here

# Option 2: Multiple keys (comma-separated) for rotation
GEMINI_API_KEYS=key1,key2,key3

# Optional: Cohere API (for reranking)
COHERE_API_KEY=your_cohere_api_key_here

# Server Configuration
PORT=5000
```

#### 6. Database Initialization

```bash
cd backend

# Create database (if using local PostgreSQL)
go run cmd/create-db/main.go

# Run migrations
go run cmd/migrate/main.go
```

The migration will create:
- `documents` table with pgvector support
- `text_search` column with GIN index for full-text search
- `chat_sessions` and `chat_messages` tables for conversation persistence

#### 7. Start Services

**Backend Server**

```bash
cd backend
go run main.go
```

Server runs on `http://localhost:5000`

**Frontend Development Server**

```bash
# From project root
npm run dev
```

Frontend runs on `http://localhost:3000`

---

## Configuration

### Environment Variables

The following environment variables can be configured in the `.env` file:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `POSTGRES_USER` | PostgreSQL username | No | `postgres` |
| `POSTGRES_PASSWORD` | PostgreSQL password | No | `postgres` |
| `POSTGRES_DB` | PostgreSQL database name | No | `rag_chatbot` |
| `DATABASE_URL` | Full PostgreSQL connection string (for manual setup) | Yes (manual) | - |
| `GEMINI_API_KEY` | Google Gemini API key (single key) | Yes | - |
| `GEMINI_API_KEYS` | Google Gemini API keys (comma-separated, for rotation) | No | - |
| `COHERE_API_KEY` | Cohere API key for reranking | No | - |
| `BACKEND_PORT` | Backend server port | No | `5000` |
| `FRONTEND_PORT` | Frontend server port | No | `3000` |
| `GIN_MODE` | Gin framework mode (debug/release) | No | `release` |

> [!CAUTION]
> Never commit the `.env` file to version control. Ensure `.env` is listed in `.gitignore`. The file contains sensitive information including API keys and database credentials.

### Hybrid Search Configuration

Hybrid search weights are configurable in `backend/handlers/chat.go`:

```go
vectorWeight := 0.7  // 70% vector, 30% text
similarityThreshold := 0.65  // Cosine distance threshold
```

**Adjustment Guidelines**:
- Higher `vectorWeight` (0.8-0.9): Better for semantic queries, synonyms
- Lower `vectorWeight` (0.3-0.5): Better for exact keyword matching
- Lower `similarityThreshold` (0.5): More strict, fewer results
- Higher `similarityThreshold` (0.7): More lenient, more results

### Chunking Parameters

Default chunking in `backend/utils/document_extractor.go`:

```go
chunkSize := 1000  // characters per chunk
overlap := 200     // characters overlap between chunks
```

Modify in `backend/utils/document_processor.go::ProcessAndSaveDocument()`.

### Model Fallback Chain

Configure in `backend/utils/chat.go`:

```go
modelsToTry := []string{
    "gemini-2.0-flash",        // Primary
    "gemini-2.0-flash-001",    // Fallback 1
    "gemini-flash-latest",     // Fallback 2
    "gemini-2.5-flash",        // Fallback 3
}
```

### OCR Configuration

OCR settings in `backend/scripts/pdf_processor.py`:

```python
matrix = fitz.Matrix(3, 3)  # Resolution scaling (3x)
threshold = 150  # Binarization threshold
custom_config = r'--oem 3 --psm 6'  # Tesseract PSM mode
```

**PSM Modes**:
- PSM 6: Single uniform block (good for tables/receipts)
- PSM 3: Fully automatic (default)
- PSM 1: Automatic with OSD

---

## Usage Guide

### Uploading Documents

#### Text Documents (PDF, TXT, DOCX)

1. Click the upload area or drag and drop files
2. Supported formats: PDF, TXT, DOCX
3. The system will automatically:
   - Extract text content
   - Perform OCR if the PDF is scanned
   - Generate embeddings
   - Store in PostgreSQL with pgvector

> [!NOTE]
> Scanned PDFs are automatically processed with Tesseract OCR. The system detects whether a PDF contains native text or requires OCR processing.

#### Structured Data (CSV, Excel)

1. Upload CSV or Excel files (.csv, .xlsx, .xls)
2. The system will:
   - Generate a file preview (columns, sample data)
   - Store metadata (no embeddings)
   - Enable Data Analyst Agent queries

### Using RAG Chat (Text Documents)

1. Select uploaded text documents from the document list
2. Type your question in natural language
3. The system will:
   - Perform hybrid search (vector + full-text)
   - Retrieve relevant chunks
   - Generate contextual response with citations
   - Stream response in real-time

**Example Queries**:
- "What is the main topic of this document?"
- "Summarize the key points"
- "What are the requirements mentioned?"

### Using Data Analyst Agent (CSV/Excel)

1. Upload a CSV or Excel file
2. Select the file in the chat interface
3. Ask analytical questions in natural language

**Example Queries**:
- "What is the average price?"
- "Show me a bar chart of sales by month"
- "What are the top 5 products by revenue?"
- "Create a scatter plot of price vs quantity"

#### Chart Visualization

The Data Analyst Agent supports automatic chart generation:

- **Supported Chart Types**: Bar charts, line charts, scatter plots, histograms, heatmaps
- **Libraries**: matplotlib and seaborn
- **Output Format**: Base64-encoded PNG images displayed inline in chat
- **Automatic Formatting**: Charts include titles, axis labels, and proper formatting

**Chart Generation Flow**:
1. User requests visualization in natural language
2. AI generates Python code with matplotlib/seaborn
3. Code is validated for security
4. Chart is generated and encoded as Base64
5. Chart is displayed in chat interface alongside AI interpretation

> [!TIP]
> When requesting charts, be specific about what you want to visualize. For example: "Create a bar chart showing sales by region" is better than "show me a chart".

---

## API Documentation

### Base URL

```
http://localhost:5000
```

### Endpoints

#### Health Check

```http
GET /ping
```

**Response:**
```json
{
  "message": "pong"
}
```

#### Upload Document

```http
POST /api/upload
Content-Type: multipart/form-data
```

**Request:**
- Form field: `document` (file: PDF, TXT, DOCX, CSV, XLSX, XLS)

**Response:**
```json
{
  "fileName": "document.pdf",
  "filePath": "uploads/document-1234567890.pdf",
  "text": "Extracted text content...",
  "message": "File berhasil diupload, divektorisasi, dan disimpan ke database (15 chunks)",
  "previewText": "First 200 characters...",
  "chunksCount": 15
}
```

#### Chat (Streaming)

```http
POST /api/chat
Content-Type: application/json
```

**Request:**
```json
{
  "question": "What is the main topic?",
  "history": [
    {
      "role": "user",
      "content": "Previous question"
    },
    {
      "role": "model",
      "content": "Previous answer"
    }
  ],
  "selectedFiles": ["document1.pdf"],
  "sessionId": 123
}
```

**Response:** Server-Sent Events (SSE) stream

**Event: `metadata`**
```json
{
  "type": "metadata",
  "sources": ["document1.pdf"],
  "sourceIds": [1, 2, 3],
  "sessionId": 123,
  "analysis": false,
  "code": null
}
```

**Event: `chunk`** (streaming)
```json
{
  "type": "chunk",
  "chunk": "Based on the uploaded documents..."
}
```

**Event: `done`**
```json
{
  "type": "done",
  "totalChunks": 15,
  "fullLength": 1234,
  "sessionId": 123
}
```

**Event: `error`**
```json
{
  "type": "error",
  "error": "Failed to generate query embedding",
  "message": "Detailed error message"
}
```

> [!NOTE]
> The endpoint automatically routes to RAG flow (PDF/TXT/DOCX) or Data Analyst flow (CSV/Excel) based on file type detection.

#### Get Documents

```http
GET /api/documents
```

**Response:**
```json
{
  "documents": ["document1.pdf", "document2.csv"],
  "count": 2
}
```

#### Delete Document

```http
DELETE /api/documents/:filename
```

**Response:**
```json
{
  "message": "Document deleted successfully",
  "deletedChunks": 15
}
```

#### Session Management

**Create Session:**
```http
POST /api/sessions
Content-Type: application/json

{
  "title": "Chat about sales data"
}
```

**Get All Sessions:**
```http
GET /api/sessions
```

**Get Session Messages:**
```http
GET /api/sessions/:id
```

**Delete Session:**
```http
DELETE /api/sessions/:id
```

---

## Project Structure

```
ai-rag-chatbot/
├── app/                          # Next.js App Router
│   ├── layout.tsx               # Root layout
│   ├── page.tsx                 # Main chat page
│   └── globals.css              # Global styles
│
├── backend/                      # Go backend
│   ├── cmd/                     # CLI utilities
│   │   ├── check-models/        # Model availability checker
│   │   ├── create-db/           # Database creation
│   │   ├── migrate/             # Migration runner
│   │   └── test-code-runner/    # Test code execution
│   │
│   ├── db/                      # Database layer
│   │   ├── db.go                # Connection pool & queries
│   │   ├── chat_store.go        # Session & message storage
│   │   └── migration*.sql       # Schema migrations
│   │
│   ├── handlers/                # HTTP handlers
│   │   ├── chat.go              # Chat endpoint (RAG + Data Analyst)
│   │   ├── upload.go            # File upload handler
│   │   ├── document.go          # Document management
│   │   ├── session.go           # Session management
│   │   └── suggestion.go        # Question suggestions
│   │
│   ├── models/                  # Data models
│   │   ├── chat.go              # Chat message struct
│   │   └── session.go           # Session struct
│   │
│   ├── routes/                  # Route definitions
│   │   └── routes.go            # Route registration
│   │
│   ├── scripts/                  # Python scripts
│   │   ├── pdf_processor.py     # PDF + OCR processing
│   │   ├── data_processor.py    # CSV/Excel to narrative
│   │   └── code_interpreter.py  # Python code execution
│   │
│   ├── utils/                   # Utility functions
│   │   ├── ai.go                # Gemini API (embeddings, chat, code gen)
│   │   ├── chat.go              # Chat generation & streaming
│   │   ├── code_runner.go       # Python execution wrapper
│   │   ├── data_preview.go      # File preview generator
│   │   ├── document_extractor.go # File extraction
│   │   ├── document_processor.go # Document processing pipeline
│   │   ├── file_helper.go       # File path resolution
│   │   ├── key_manager.go       # API key rotation
│   │   └── reranker.go          # Cohere reranking
│   │
│   ├── Dockerfile               # Docker image for Go + Python + Tesseract
│   └── main.go                  # Application entry point
│
├── components/                   # React components
│   ├── chat/                    # Chat components
│   │   ├── ChatBubble.tsx       # Message bubble
│   │   ├── ChatContainer.tsx    # Chat container
│   │   ├── ChatInput.tsx        # Input with attachment upload
│   │   └── TypingIndicator.tsx  # Loading indicator
│   │
│   ├── layout/                  # Layout components
│   │   └── Sidebar.tsx          # Session sidebar
│   │
│   ├── upload/                  # Upload components
│   │   ├── UploadCard.tsx       # Upload interface
│   │   └── DocumentList.tsx     # Document list
│   │
│   └── ui/                      # UI primitives
│       ├── PDFViewerPanel.tsx  # PDF viewer
│       └── ConfirmDialog.tsx   # Confirmation dialog
│
├── my-app/                       # Frontend Next.js
│   └── Dockerfile               # Docker image for Next.js
│
├── public/                       # Static assets
├── package.json                  # Frontend dependencies
├── tailwind.config.js           # Tailwind configuration
├── docker-compose.yml            # Docker Compose orchestration
├── .dockerignore                 # Docker ignore patterns
└── README.md                     # This file
```

---

## Development

### Running in Development

#### Option 1: Docker Compose

```bash
# Start all services
docker-compose up --build

# Start in background
docker-compose up -d --build

# View logs
docker-compose logs -f backend
docker-compose logs -f frontend
```

#### Option 2: Manual Development

**Backend**

```bash
cd backend
go run main.go
```

**Frontend**

```bash
npm run dev
```

### Utility Commands

#### Check Gemini Models

```bash
cd backend
go run cmd/check-models/main.go
```

#### Database Operations

```bash
cd backend

# Create database
go run cmd/create-db/main.go

# Run migrations
go run cmd/migrate/main.go
```

### Code Style

- **Go**: Standard Go conventions, use `gofmt`
- **TypeScript/React**: ESLint configuration included
- **Python**: PEP 8 style guide

### Adding New Features

1. **New API Endpoint**: Add handler in `handlers/`, register in `routes/routes.go`
2. **New UI Component**: Add to `components/` directory
3. **Database Changes**: Create migration file in `db/`
4. **Python Script**: Add to `scripts/` with proper error handling

---

## Troubleshooting

### Docker Issues

**Error**: `Cannot connect to Docker daemon`

**Solutions**:
- Ensure Docker Desktop is running (Windows/Mac)
- Verify Docker service is active: `docker ps`
- Check Docker Compose version: `docker-compose --version`

**Error**: `Service 'backend' failed to build`

**Solutions**:
- Check Dockerfile syntax
- Verify Go version compatibility (1.24+)
- Check Python dependencies installation in Dockerfile
- Review build logs: `docker-compose build --no-cache backend`

**Error**: `Port already in use`

**Solutions**:
- Change ports in `docker-compose.yml` or `.env`
- Stop conflicting services: `docker-compose down`
- Check port usage: `netstat -ano | findstr :5000` (Windows) or `lsof -i :5000` (Linux/Mac)

**Error**: `Python script not found` in Docker container

**Solutions**:
- Verify scripts are copied: `docker-compose exec backend ls -la /app/scripts/`
- Rebuild backend: `docker-compose build --no-cache backend`
- Check Dockerfile COPY commands

### Database Connection Failed

**Error**: `connection timeout expired`

**Solutions**:
- Verify PostgreSQL is running: `docker ps` or service status
- Check `DATABASE_URL` in `.env` file
- Ensure pgvector extension: `CREATE EXTENSION vector;`
- Verify port (default: 5433 for Docker, 5432 for local)

### Hybrid Search Returns No Results

**Issue**: Hybrid search yields 0 results

**Solutions**:
- System automatically falls back to vector-only search
- Check logs for fallback messages
- Verify GIN index exists: `\d documents` in psql
- Check if `text_search` column is populated
- Adjust similarity threshold if needed

### Tesseract OCR Not Found

**Error**: `Tesseract OCR tidak ditemukan`

**Solutions**:
- Install Tesseract from [GitHub](https://github.com/UB-Mannheim/tesseract/wiki)
- Verify installation path matches auto-detection
- Add Tesseract to PATH environment variable
- Download language data (eng + ind)

> [!NOTE]
> If using Docker Compose, Tesseract OCR is pre-installed in the backend image. This error only occurs in manual installation.

### Python Code Execution Fails

**Error**: `failed to execute Python code`

**Solutions**:
- Verify Python is installed: `python --version`
- Install required packages: `pip install pandas openpyxl`
- Check file path is correct
- Review code validation errors in logs

### API Key Issues

**Error**: `Invalid API key` or `rate limit`

**Solutions**:
- Verify API key in `.env` file
- Use `GEMINI_API_KEYS` for multiple keys (comma-separated)
- System automatically rotates keys on rate limit
- Check API key quota in Google Cloud Console

### Model Not Found

**Error**: `models/gemini-2.0-flash is not found`

**Solutions**:
- Run model checker: `go run cmd/check-models/main.go`
- Update model name in `utils/chat.go` to available model
- Fallback chain will try alternative models automatically

---

## License

[License information]

---

## Acknowledgments

- Google Gemini AI for embeddings and generation models
- pgvector for PostgreSQL vector similarity search
- Cohere for document reranking
- Tesseract OCR for document scanning support
- Next.js and React teams for frontend frameworks
- Gin framework for Go web development
