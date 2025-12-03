# Multi-stage build untuk Go + Python + Tesseract OCR
FROM golang:1.24 AS builder

# Install Python 3, pip, dan Tesseract OCR
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-ind \
    libtesseract-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies untuk Data Analyst Agent dan OCR
RUN pip3 install --no-cache-dir --break-system-packages \
    pandas \
    openpyxl \
    pymupdf \
    pytesseract \
    google-generativeai \
    pillow \
    matplotlib \
    seaborn \
    numpy

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies: Python 3, Tesseract OCR
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-ind \
    libtesseract-dev \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies (runtime)
RUN pip3 install --no-cache-dir --break-system-packages \
    pandas \
    openpyxl \
    pymupdf \
    pytesseract \
    google-generativeai \
    pillow \
    matplotlib \
    seaborn \
    numpy

# Set working directory
WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/main .

# Copy Python scripts from builder
COPY --from=builder /app/scripts/ ./scripts/

# Create uploads directory
RUN mkdir -p uploads

# Expose port
EXPOSE 5000

# Run application
CMD ["./main"]

