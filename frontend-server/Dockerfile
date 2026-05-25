# ============================================================
# frontend-server/Dockerfile  (Production)
# ============================================================
# Multi-stage: tidak perlu, Flask sangat ringan.
# Base: python:3.12-slim → copy app + templates → gunicorn.
#
# Build context: ./frontend-server/
# Usage (dari docker-compose.prod.yml):
#   build:
#     context: ../frontend-server
#     dockerfile: Dockerfile
# ============================================================

FROM python:3.12-slim

WORKDIR /app

# Install dependencies dulu (layer cache-friendly)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy seluruh source Flask
COPY . .

# Expose port gunicorn
EXPOSE 5000

# Production: jalankan dengan gunicorn, 2 worker
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "2", "--timeout", "60", "app:app"]
