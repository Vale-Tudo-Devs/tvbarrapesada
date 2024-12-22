# Build stage
FROM node:22.11-bookworm-slim

WORKDIR /app

# Install pnpm globally
RUN npm install pnpm -g

# Copy package files first
COPY package.json package-lock.json ./

# Install dependencies
RUN pnpm install

# Copy rest of the files
COPY . /

# Copy TypeScript config
COPY tsconfig.json ./

# Build TypeScript
RUN pnpm run build

# Install ffmpeg
RUN apt update && apt install -y ffmpeg

# Start application
CMD ["pnpm", "run", "start"]