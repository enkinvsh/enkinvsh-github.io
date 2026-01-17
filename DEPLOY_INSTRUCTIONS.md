# Focus Backend — Production Deployment Guide

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Internet                              │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    Caddy (Port 80/443)                       │
│              Automatic HTTPS via Let's Encrypt               │
│                     HTTP/3 (QUIC) Support                    │
└─────────────────────────┬───────────────────────────────────┘
                          │ reverse_proxy
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Focus API (Port 8080)                      │
│                     Go 1.25 + Gin                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                 PostgreSQL 16 (Port 5432)                    │
│                   Internal network only                      │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- VPS with Ubuntu 22.04+ / Debian 12+
- Domain pointing to VPS IP (A record)
- Docker & Docker Compose installed
- Ports 80, 443 open in firewall

---

## Step 1: Prepare VPS

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Create project directory
sudo mkdir -p /opt/focus
sudo chown $USER:$USER /opt/focus
cd /opt/focus
```

## Step 2: Configure DNS

Point your domain to your VPS:

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | api.focus | YOUR_VPS_IP | 300 |

Wait for DNS propagation (5-30 minutes).

## Step 3: Create Environment File

```bash
cd /opt/focus

cat > .env << 'EOF'
# Database
DB_PASSWORD=GENERATE_SECURE_PASSWORD_HERE

# Telegram Bot
BOT_TOKEN=YOUR_BOT_TOKEN_FROM_BOTFATHER

# Google Gemini
GEMINI_KEY=YOUR_GEMINI_API_KEY

# Domain (without https://)
DOMAIN=api.focus.yourdomain.com
EOF

# Generate secure password
sed -i "s/GENERATE_SECURE_PASSWORD_HERE/$(openssl rand -base64 32 | tr -d '=/+')/" .env

# Secure the file
chmod 600 .env
```

## Step 4: Deploy Stack

### Option A: Using Pre-built Image (Recommended)

```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Create docker-compose.prod.yml
cat > docker-compose.yml << 'EOF'
services:
  postgres:
    image: postgres:16-alpine
    container_name: focus-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: focus
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: focus
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - internal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U focus -d focus"]
      interval: 10s
      timeout: 5s
      retries: 5

  app:
    image: ghcr.io/YOUR_USERNAME/focus-backend:latest
    container_name: focus-api
    restart: unless-stopped
    environment:
      DATABASE_URL: postgres://focus:${DB_PASSWORD}@postgres:5432/focus?sslmode=disable
      BOT_TOKEN: ${BOT_TOKEN}
      GEMINI_KEY: ${GEMINI_KEY}
      PORT: 8080
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - internal
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  caddy:
    image: caddy:2-alpine
    container_name: focus-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    environment:
      DOMAIN: ${DOMAIN}
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - app
    networks:
      - internal

networks:
  internal:

volumes:
  pgdata:
  caddy_data:
  caddy_config:
EOF

# Create Caddyfile
cat > Caddyfile << 'EOF'
{$DOMAIN} {
    reverse_proxy app:8080
    encode gzip zstd
}
EOF

# Start services
docker compose up -d
```

### Option B: Build from Source

```bash
# Clone repository
git clone https://github.com/YOUR_USERNAME/focus.git /opt/focus/src
cd /opt/focus/src/backend

# Copy .env
cp /opt/focus/.env .env

# Build and start
docker compose up -d --build
```

## Step 5: Set Telegram Webhook

After deployment, configure the webhook:

```bash
# Replace with your actual values
BOT_TOKEN="your_bot_token"
DOMAIN="api.focus.yourdomain.com"

curl -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"https://${DOMAIN}/bot/webhook\"}"
```

Expected response:
```json
{"ok":true,"result":true,"description":"Webhook was set"}
```

Verify webhook:
```bash
curl "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo"
```

## Step 6: Verify Deployment

```bash
# Check all containers are running
docker compose ps

# Check API health
curl https://api.focus.yourdomain.com/health

# Check logs
docker compose logs -f

# Check SSL certificate
curl -vI https://api.focus.yourdomain.com 2>&1 | grep -i "SSL\|certificate"
```

---

## GitHub Actions: Automated Deployment

### Required Secrets

Add these in **Repository → Settings → Secrets → Actions**:

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | VPS IP or domain |
| `VPS_USER` | SSH username (root or deploy user) |
| `VPS_SSH_KEY` | Private SSH key (ed25519 recommended) |
| `VPS_PORT` | SSH port (default: 22) |

### Generate Deploy Key

```bash
# On your local machine
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/focus_deploy -N ""

# Copy public key to VPS
ssh-copy-id -i ~/.ssh/focus_deploy.pub user@YOUR_VPS_IP

# Copy private key content to GitHub Secret VPS_SSH_KEY
cat ~/.ssh/focus_deploy
```

### Workflow

The `.github/workflows/deploy.yml` automatically:
1. Builds Docker image on push to `main`
2. Pushes to GitHub Container Registry
3. SSHs to VPS and pulls latest image
4. Restarts the container

---

## Maintenance

### Update Application

```bash
cd /opt/focus
docker compose pull
docker compose up -d
docker image prune -f
```

### Backup Database

```bash
docker exec focus-db pg_dump -U focus focus > backup_$(date +%Y%m%d).sql
```

### Restore Database

```bash
cat backup.sql | docker exec -i focus-db psql -U focus focus
```

### View Logs

```bash
docker compose logs -f app      # API logs
docker compose logs -f caddy    # Caddy/SSL logs
docker compose logs -f postgres # Database logs
```

### Restart Services

```bash
docker compose restart          # Restart all
docker compose restart app      # Restart API only
```

---

## Troubleshooting

### SSL Certificate Issues

```bash
# Check Caddy logs
docker compose logs caddy

# Force certificate renewal
docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile
```

### Database Connection Issues

```bash
# Check if PostgreSQL is healthy
docker compose exec postgres pg_isready -U focus

# Connect to database
docker compose exec postgres psql -U focus focus
```

### Webhook Not Working

```bash
# Check webhook status
curl "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo"

# Check API logs for incoming requests
docker compose logs -f app | grep webhook
```

### Container Won't Start

```bash
# Check detailed logs
docker compose logs --tail=100 app

# Check container status
docker inspect focus-api | jq '.[0].State'
```

---

## Security Checklist

- [ ] Strong `DB_PASSWORD` (32+ random characters)
- [ ] `.env` file has `chmod 600`
- [ ] SSH key-only authentication enabled
- [ ] Firewall allows only 22, 80, 443
- [ ] Regular backups configured
- [ ] Monitoring/alerting set up

---

## File Structure on VPS

```
/opt/focus/
├── .env              # Secrets (chmod 600)
├── docker-compose.yml
├── Caddyfile
└── backups/          # Database backups
```
