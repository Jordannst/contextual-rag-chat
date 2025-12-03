# Multi-stage build untuk Next.js
FROM node:20-alpine AS deps

# Install dependencies only when needed
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies with increased timeout for slow connections
RUN npm config set fetch-retry-mintimeout 20000 && \
    npm config set fetch-retry-maxtimeout 120000 && \
    npm ci

# Builder stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy source code
COPY . .

# Build Next.js application
RUN npm run build

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV production

# Copy package files
COPY package.json ./

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy built application
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next

# Expose port
EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

# Start application
CMD ["npm", "start"]

