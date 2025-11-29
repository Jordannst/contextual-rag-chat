# 🤖 AI RAG Chatbot

<div align="center">

![Next.js](https://img.shields.io/badge/Next.js-16.0-black?style=for-the-badge&logo=next.js)
![React](https://img.shields.io/badge/React-19.2-blue?style=for-the-badge&logo=react)
![Go](https://img.shields.io/badge/Go-1.24-blue?style=for-the-badge&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue?style=for-the-badge&logo=postgresql)
![Gemini](https://img.shields.io/badge/Google%20Gemini-2.0-orange?style=for-the-badge&logo=google)

**A modern, production-ready Retrieval-Augmented Generation (RAG) chatbot powered by Google Gemini AI**

[Features](#-features) • [Tech Stack](#-tech-stack) • [Quick Start](#-quick-start) • [Architecture](#-architecture) • [API Documentation](#-api-documentation)

</div>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Architecture](#-architecture)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
- [API Documentation](#-api-documentation)
- [Project Structure](#-project-structure)
- [Development](#-development)
- [Troubleshooting](#-troubleshooting)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🎯 Overview

**AI RAG Chatbot** is a sophisticated conversational AI application that combines the power of large language models with semantic document search. Upload PDF or text documents, and have intelligent conversations about their content using Google's Gemini AI models.

### Key Capabilities

- 📄 **Document Upload & Processing**: Upload PDF or TXT files with automatic text extraction
- 🔍 **Semantic Search**: Find relevant document chunks using vector similarity search
- 💬 **Context-Aware Chat**: Ask questions and get answers based on your uploaded documents
- 🧠 **Conversation History**: Maintains context across multiple messages
- ⚡ **Smart Chunking**: Intelligent text splitting with overlap for better context preservation
- 🎨 **Modern UI**: Beautiful, responsive interface inspired by Google's Gemini design

---

## ✨ Features

### 🚀 Core Features

- **Document Processing**
  - Support for PDF and TXT files
  - Automatic text extraction and chunking
  - Smart sentence boundary detection
  - Configurable chunk size and overlap

- **Vector Storage & Search**
  - PostgreSQL with pgvector extension
  - 768-dimensional embeddings using Google Gemini
  - Cosine similarity search for semantic matching
  - Source file tracking for each document chunk

- **RAG Pipeline**
  - Query embedding generation
  - Top-K similar document retrieval
  - Context-aware response generation
  - Automatic fallback model chain

- **Conversation Management**
  - Full conversation history support
  - Context preservation across messages
  - Multi-turn dialog handling

### 🎨 User Interface

- **Modern Design**
  - Dark mode optimized UI
  - Smooth animations and transitions
  - Responsive layout for all devices
  - Typing indicators for better UX

- **Interactive Components**
  - Drag-and-drop file upload
  - Real-time chat interface
  - Prompt suggestions
  - Message timestamps

### 🔧 Developer Experience

- **Comprehensive Logging**
  - Step-by-step request tracking
  - Detailed error messages
  - Performance monitoring

- **Utility Tools**
  - Model checker script
  - Database migration tools
  - Environment variable management with BOM handling

---

## 🛠 Tech Stack

### Frontend

| Technology | Version | Purpose |
|------------|---------|---------|
| **Next.js** | 16.0.5 | React framework with App Router |
| **React** | 19.2.0 | UI library |
| **TypeScript** | ^5 | Type-safe JavaScript |
| **Tailwind CSS** | ^4 | Utility-first CSS framework |
| **React Markdown** | ^10.1.0 | Markdown rendering for responses |

### Backend

| Technology | Version | Purpose |
|------------|---------|---------|
| **Go** | 1.24.1 | High-performance backend language |
| **Gin** | 1.9.1 | HTTP web framework |
| **PostgreSQL** | 16+ | Relational database |
| **pgvector** | Latest | Vector similarity search extension |
| **Google Gemini AI** | 2.0 | Embeddings and text generation |
| **pgx** | v5 | PostgreSQL driver for Go |

### AI & ML

- **Embedding Model**: `text-embedding-004` (768 dimensions)
- **Generative Model**: `gemini-2.0-flash` with automatic fallback chain
  - Primary: `gemini-2.0-flash`
  - Fallbacks: `gemini-2.0-flash-001` → `gemini-flash-latest` → `gemini-2.5-flash`

### Infrastructure

- **Document Processing**: `github.com/ledongthuc/pdf` for PDF extraction
- **Environment**: `godotenv` with BOM handling
- **CORS**: Enabled for cross-origin requests

---

## 🏗 Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (Next.js)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Chat UI    │  │ File Upload  │  │   Sidebar    │     │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘     │
└─────────┼──────────────────┼──────────────────────────────┘
          │                  │
          │ HTTP/REST API    │
          │                  │
┌─────────┼──────────────────┼──────────────────────────────┐
│         ▼                  ▼                               │
│                  Backend (Go/Gin)                          │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  API Handlers                                       │  │
│  │  - /api/chat                                        │  │
│  │  - /api/upload                                      │  │
│  └──────┬──────────────────────┬──────────────────────┘  │
│         │                      │                          │
│  ┌──────▼──────────┐  ┌────────▼─────────────┐          │
│  │  RAG Pipeline   │  │  Document Processor  │          │
│  │  1. Embed Query │  │  1. Extract Text     │          │
│  │  2. Vector Search│  │  2. Split Chunks    │          │
│  │  3. Generate    │  │  3. Generate Embed   │          │
│  │     Response    │  │  4. Store in DB      │          │
│  └──────┬──────────┘  └────────┬─────────────┘          │
│         │                      │                          │
└─────────┼──────────────────────┼──────────────────────────┘
          │                      │
          │                      │
┌─────────▼──────────────────────▼──────────────────────────┐
│              External Services                             │
│  ┌──────────────┐              ┌──────────────┐          │
│  │ Google Gemini│              │  PostgreSQL  │          │
│  │ - Embeddings │              │  + pgvector  │          │
│  │ - Chat       │              │              │          │
│  └──────────────┘              └──────────────┘          │
└───────────────────────────────────────────────────────────┘
```

### RAG Pipeline Flow

1. **Document Upload**
   ```
   PDF/TXT → Extract Text → Split into Chunks → Generate Embeddings → Store in DB
   ```

2. **Query Processing**
   ```
   User Question → Generate Query Embedding → Vector Similarity Search → 
   Retrieve Top-K Chunks → Build Context → Generate Response with Gemini → Return Answer
   ```

3. **Conversation Context**
   ```
   Current Question + History + Retrieved Documents → Contextual Prompt → 
   Gemini Response → Update History
   ```

---

## 📦 Prerequisites

Before you begin, ensure you have the following installed:

- **Node.js** 18+ and npm
- **Go** 1.24+ ([Download](https://go.dev/dl/))
- **PostgreSQL** 16+ with pgvector extension ([Installation Guide](https://github.com/pgvector/pgvector))
- **Google Gemini API Key** ([Get one here](https://makersuite.google.com/app/apikey))

### PostgreSQL Setup

You can use either:
- **Local PostgreSQL** with pgvector extension
- **Docker** (recommended for quick setup):
  ```bash
  docker run -d \
    --name rag-chatbot-postgres \
    -p 5433:5432 \
    -e POSTGRES_USER=postgres \
    -e POSTGRES_PASSWORD=your_password \
    -e POSTGRES_DB=rag_chatbot \
    pgvector/pgvector:pg16
  ```

---

## 🚀 Installation

### 1. Clone the Repository

```bash
git clone <your-repo-url>
cd ai-rag-chatbot/my-app
```

### 2. Frontend Setup

```bash
# Install dependencies
npm install

# The frontend is ready! (No build step needed for dev)
```

### 3. Backend Setup

```bash
cd backend

# Install Go dependencies
go mod download

# Build the application
go build -o backend.exe main.go
```

### 4. Environment Configuration

Create a `.env` file in the `backend/` directory:

```env
# Database Configuration
DATABASE_URL=postgresql://postgres:your_password@localhost:5433/rag_chatbot

# Google Gemini API
GEMINI_API_KEY=your_gemini_api_key_here

# Server Configuration
PORT=5000
```

### 5. Database Setup

```bash
cd backend

# Create the database (if using local PostgreSQL)
go run cmd/create-db/main.go

# Run migrations
go run cmd/migrate/main.go
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string | ✅ Yes | - |
| `GEMINI_API_KEY` | Google Gemini API key | ✅ Yes | - |
| `PORT` | Backend server port | ❌ No | `5000` |

### Chunking Configuration

Default chunking parameters (in `handlers/upload.go`):
- **Chunk Size**: 1000 characters
- **Overlap**: 200 characters

To modify, update the `SplitText` call:
```go
chunks := utils.SplitText(text, 1000, 200) // (text, chunkSize, overlap)
```

### Model Configuration

The chatbot uses a fallback chain for reliability. To modify models, edit `backend/utils/chat.go`:

```go
modelsToTry := []string{
    "gemini-2.0-flash",        // Primary
    "gemini-2.0-flash-001",    // Fallback 1
    "gemini-flash-latest",     // Fallback 2
    "gemini-2.5-flash",        // Fallback 3
}
```

---

## 🎮 Usage

### Starting the Application

#### 1. Start the Backend

```bash
cd backend
go run main.go

# Or using the compiled binary
./backend.exe
```

The backend will start on `http://localhost:5000`

#### 2. Start the Frontend

```bash
# From the root directory
npm run dev
```

The frontend will be available at `http://localhost:3000`

### Using the Chatbot

1. **Upload a Document**
   - Click the upload area or drag-and-drop a PDF/TXT file
   - Wait for the document to be processed (extracted, chunked, and indexed)

2. **Start Chatting**
   - Type your question in the chat input
   - The AI will search your documents and provide contextual answers
   - Continue the conversation - context is preserved across messages

3. **Use Prompt Suggestions**
   - Click on suggested prompts to get started quickly
   - Examples: "Plan a trip", "Explain a concept", etc.

---

## 📚 API Documentation

### Base URL

```
http://localhost:5000
```

### Endpoints

#### 1. Health Check

```http
GET /ping
```

**Response:**
```json
{
  "message": "pong"
}
```

---

#### 2. Upload Document

```http
POST /api/upload
Content-Type: multipart/form-data
```

**Request:**
- Form field: `document` (file: PDF or TXT)

**Response:**
```json
{
  "fileName": "example.pdf",
  "filePath": "uploads/example-1234567890.pdf",
  "text": "Extracted text content...",
  "message": "File berhasil diupload, divektorisasi, dan disimpan ke database (5 chunks)",
  "previewText": "First 200 characters...",
  "chunksCount": 5,
  "totalChunks": 5
}
```

**Error Response:**
```json
{
  "error": "Only PDF and TXT files are allowed"
}
```

---

#### 3. Chat

```http
POST /api/chat
Content-Type: application/json
```

**Request:**
```json
{
  "question": "What is the main topic of the document?",
  "history": [
    {
      "role": "user",
      "content": "Previous question"
    },
    {
      "role": "model",
      "content": "Previous answer"
    }
  ]
}
```

**Response:**
```json
{
  "response": "Based on the uploaded documents, the main topic is...",
  "sources": [
    "Document chunk 1 content...",
    "Document chunk 2 content...",
    "Document chunk 3 content..."
  ],
  "sourceIds": [1, 2, 3]
}
```

**Error Response:**
```json
{
  "error": "Failed to generate query embedding",
  "message": "Detailed error message"
}
```

---

## 📁 Project Structure

```
ai-rag-chatbot/
├── app/                        # Next.js App Router
│   ├── layout.tsx             # Root layout
│   ├── page.tsx               # Main chat page
│   └── globals.css            # Global styles
│
├── backend/                    # Go backend
│   ├── cmd/                   # CLI tools
│   │   ├── check-models/      # Model checker utility
│   │   ├── create-db/         # Database creation
│   │   └── migrate/           # Migration runner
│   │
│   ├── db/                    # Database layer
│   │   ├── db.go              # Connection & queries
│   │   └── migration.sql      # Schema definition
│   │
│   ├── handlers/              # HTTP handlers
│   │   ├── chat.go            # Chat endpoint
│   │   └── upload.go          # Upload endpoint
│   │
│   ├── models/                # Data models
│   │   └── chat.go            # Chat message struct
│   │
│   ├── routes/                # Route definitions
│   │   └── routes.go          # Route registration
│   │
│   ├── utils/                 # Utility functions
│   │   ├── ai.go              # Gemini API client
│   │   ├── chat.go            # Chat generation logic
│   │   ├── document_extractor.go  # PDF/TXT extraction
│   │   └── env.go             # Environment handling
│   │
│   ├── main.go                # Application entry point
│   ├── go.mod                 # Go dependencies
│   └── scripts.ps1            # Development scripts
│
├── components/                 # React components
│   ├── chat/                  # Chat components
│   ├── layout/                # Layout components
│   ├── ui/                    # UI primitives
│   └── upload/                # Upload components
│
├── public/                     # Static assets
├── package.json               # Frontend dependencies
├── tailwind.config.js         # Tailwind configuration
└── README.md                  # This file
```

---

## 🔨 Development

### Running in Development Mode

#### Backend

```bash
cd backend
go run main.go
```

#### Frontend

```bash
npm run dev
```

### Utility Commands

#### Check Available Gemini Models

```bash
cd backend
go run cmd/check-models/main.go
```

This will list all available models for your API key and show recommendations.

#### Database Operations

```bash
# Create database
cd backend
go run cmd/create-db/main.go

# Run migrations
go run cmd/migrate/main.go
```

### Code Style

- **Go**: Follow standard Go conventions, use `gofmt`
- **TypeScript/React**: ESLint configuration included
- **Formatting**: Prettier recommended for frontend

### Adding New Features

1. **New API Endpoint**: Add handler in `handlers/`, register in `routes/routes.go`
2. **New UI Component**: Add to `components/` directory
3. **Database Changes**: Create new migration file in `db/`

---

## 🐛 Troubleshooting

### Common Issues

#### 1. Database Connection Failed

**Error**: `connection timeout expired`

**Solution**:
- Verify PostgreSQL is running: `docker ps` or check service status
- Check `DATABASE_URL` in `.env` file
- Ensure pgvector extension is installed: `CREATE EXTENSION vector;`

#### 2. Model Not Found (404)

**Error**: `models/gemini-1.5-flash is not found`

**Solution**:
- Run model checker: `go run cmd/check-models/main.go`
- Update model name in `utils/chat.go` to a valid model
- The fallback chain will automatically try alternative models

#### 3. Environment Variables Not Loading

**Error**: `DATABASE_URL is not set`

**Solution**:
- Ensure `.env` file exists in `backend/` directory
- Check for BOM (Byte Order Mark) in `.env` file - the app handles this automatically
- Verify file format: `KEY=value` (no quotes needed)

#### 4. PDF Extraction Fails

**Error**: `failed to extract text from PDF`

**Solution**:
- Ensure PDF is not password-protected
- Check if PDF contains text (not just images)
- Verify file is valid PDF format
- Check logs for detailed error messages

#### 5. Port Already in Use

**Error**: `bind: address already in use`

**Solution**:
- Find process using port: `netstat -ano | findstr :5000` (Windows)
- Kill process or change PORT in `.env`

### Debug Mode

Enable detailed logging by checking the console output. The backend logs every step of the RAG pipeline:

```
[Chat] Step 1: Request diterima
[Chat] Step 2: Generating embedding...
[Chat] Step 3: Mencari dokumen di DB...
[Chat] Step 4: Dokumen ditemukan: 3 dokumen
[Chat] Step 5: Mengirim prompt ke Gemini...
```

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write clear commit messages following [Conventional Commits](https://www.conventionalcommits.org/)
- Add tests for new features
- Update documentation as needed
- Ensure code passes linting/formatting checks

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Google Gemini AI](https://ai.google.dev/) for powerful language models
- [pgvector](https://github.com/pgvector/pgvector) for vector similarity search
- [Next.js](https://nextjs.org/) and [React](https://react.dev/) teams for amazing frameworks
- [Gin](https://gin-gonic.com/) for the elegant Go web framework

---

## 📞 Support

For questions, issues, or feature requests, please open an issue on GitHub.

---

<div align="center">

**Built with ❤️ using Next.js, Go, and Google Gemini AI**

⭐ Star this repo if you find it helpful!

</div>
