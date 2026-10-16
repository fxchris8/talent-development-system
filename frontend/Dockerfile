# ============================================
# Development Dockerfile for Next.js
# This uses bind mounts from docker-compose
# ============================================
FROM node:20-alpine

WORKDIR /app

# Install dependencies for development
RUN apk add --no-cache libc6-compat

# Copy package files
COPY package.json package-lock.json* ./

# Install all dependencies (including devDependencies for hot reload, skip husky)
RUN npm ci --ignore-scripts

# Expose port
EXPOSE 3000

# Development server with hot reload.
# Turbopack does not notice file changes on a Windows/macOS bind mount, so inside Docker the
# webpack dev server is used: it honours WATCHPACK_POLLING (set in docker-compose.yml).
CMD ["npx", "next", "dev", "--webpack"]
