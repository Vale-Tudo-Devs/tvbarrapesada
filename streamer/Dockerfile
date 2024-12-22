# Build stage
FROM node:18 AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies including TypeScript
RUN npm ci && npm cache clean --force

# Copy source code
COPY tsconfig.json ./
COPY ./src ./src

# Build TypeScript
RUN npm run build

# Production stage
FROM node:18

RUN apt update && apt install -y ffmpeg
WORKDIR /app

COPY package*.json ./
RUN npm ci --only=production

# Copy built files from builder
COPY --from=builder /app/dist ./dist

EXPOSE 3000
CMD [ "node", "dist/index.ts" ]