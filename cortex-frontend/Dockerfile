## Multi-stage build for Cortex Web UI (Angular)

FROM node:20-alpine AS builder
WORKDIR /app

# Install dependencies in a cache-friendly layer
COPY package*.json ./
RUN npm install --no-audit --no-fund

# Copy source and build for production
COPY . .
RUN npm run build -- --configuration production


FROM nginx:alpine AS final
WORKDIR /usr/share/nginx/html

# Copy SPA build output
COPY --from=builder /app/dist/cortex-frontend/browser ./

# Nginx config for SPA routing + API proxy
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

