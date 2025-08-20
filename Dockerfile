# --- Build Stage ---
FROM node:lts AS builder

WORKDIR /app

# Install frontend dependencies and build
COPY frontend/package*.json ./frontend/
RUN npm install --prefix frontend
COPY frontend/ ./frontend/
RUN npm run build --prefix frontend

# Install backend dependencies and build
COPY backend/package*.json ./backend/
RUN npm install --prefix backend
COPY backend/ ./backend/
RUN npm run build --prefix backend

# Prune dev dependencies from backend node_modules
RUN npm prune --prefix backend --omit=dev


# --- Backend Runtime Stage with Frontend ---
FROM gcr.io/distroless/nodejs20-debian12 AS runtime

WORKDIR /app

# Copy built app and package files from builder
COPY --from=builder /app/backend/dist ./backend/dist
COPY --from=builder /app/backend/package*.json ./backend/
COPY --from=builder /app/backend/node_modules ./backend/node_modules
COPY --from=builder /app/frontend/dist ./public

# Expose backend port
EXPOSE 3001

# Command to start backend server
# Distroless images have a non-root user and a default entrypoint.
# We just need to provide the path to our script.
CMD ["backend/dist/index.js"]