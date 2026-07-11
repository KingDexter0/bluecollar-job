# Staging Deployment Guide: Linode VPS + Docker Compose

This guide deploys the staging build on a Linode VPS using Docker Compose, NGINX, and Certbot.

## 1. Server Setup

Create an Ubuntu 22.04 or 24.04 Linode VPS.

```bash
ssh root@YOUR_SERVER_IP
apt update && apt upgrade -y
adduser deploy
usermod -aG sudo deploy
rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy
su - deploy
```

## 2. Docker Installation

```bash
sudo apt update
sudo apt install -y ca-certificates curl gnupg git ufw nginx certbot python3-certbot-nginx python3
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker deploy
```

Log out and back in so the `docker` group applies.

## 3. Firewall Setup

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

Do not expose PostgreSQL or Redis publicly. Use managed services or private-network hosts.

## 4. Domain Setup

Create DNS records:

- `staging.example.com` -> VPS public IP
- `api-staging.example.com` -> VPS public IP

Wait for DNS propagation before requesting SSL.

## 5. Clone And Configure

```bash
git clone https://github.com/KingDexter0/bluecollar-job.git
cd bluecollar-job
cp .env.staging.example .env.staging
nano .env.staging
```

Set real staging values:

- `DATABASE_URL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `ADMIN_TOKEN`
- `CORS_ALLOWED_ORIGINS`
- `FRONTEND_URL`
- `WHATSAPP_PROVIDER`
- `WHATSAPP_VERIFY_TOKEN`
- `WHATSAPP_ACCESS_TOKEN` when `WHATSAPP_PROVIDER=meta`
- `WHATSAPP_PHONE_NUMBER_ID` when `WHATSAPP_PROVIDER=meta`
- `WHATSAPP_BUSINESS_ACCOUNT_ID`
- `WHATSAPP_GRAPH_API_VERSION`
- `NEXT_PUBLIC_API_BASE_URL`

For staging without document upload, keep:

```env
DOCUMENT_UPLOAD_ENABLED=false
```

## 6. NGINX Reverse Proxy

Copy the example config:

```bash
sudo cp deploy/nginx/staging.conf.example /etc/nginx/sites-available/bluecollar-staging
sudo nano /etc/nginx/sites-available/bluecollar-staging
sudo ln -s /etc/nginx/sites-available/bluecollar-staging /etc/nginx/sites-enabled/bluecollar-staging
sudo nginx -t
sudo systemctl reload nginx
```

The config routes:

- frontend domain to `127.0.0.1:3000`
- API domain to `127.0.0.1:8081`
- WhatsApp webhook path with proxy headers and request ID forwarding

## 7. SSL With Certbot

```bash
sudo certbot --nginx -d staging.example.com -d api-staging.example.com
sudo certbot renew --dry-run
```

## 8. Deploy

```bash
chmod +x scripts/deploy-staging.sh scripts/smoke-test.sh scripts/rollback-staging.sh
ENV_FILE=.env.staging SMOKE_BASE_URL=https://api-staging.example.com ./scripts/deploy-staging.sh
```

Manual equivalent:

```bash
docker compose --env-file .env.staging -f docker-compose.prod.yml build
docker compose --env-file .env.staging -f docker-compose.prod.yml run --rm api /app/migrate up
docker compose --env-file .env.staging -f docker-compose.prod.yml up -d
./scripts/smoke-test.sh https://api-staging.example.com
```

## 9. Health Checks

```bash
curl https://api-staging.example.com/health
curl https://api-staging.example.com/ready
curl https://api-staging.example.com/live
curl https://api-staging.example.com/metrics
```

## 10. Rollback

Rollback to the previous commit:

```bash
ENV_FILE=.env.staging HEALTH_URL=https://api-staging.example.com/health ./scripts/rollback-staging.sh HEAD~1
```

Rollback to a tag or specific commit:

```bash
ENV_FILE=.env.staging ./scripts/rollback-staging.sh v2026.07.04-staging
```

After rollback:

```bash
docker compose --env-file .env.staging -f docker-compose.prod.yml ps
curl https://api-staging.example.com/ready
```

## 11. Backup And Restore

Run logical backups from a secure operator machine or the VPS:

```bash
DATABASE_URL="postgres://..." powershell.exe -NoProfile -ExecutionPolicy Bypass -File ./scripts/backup-postgres.ps1
```

Restore only into staging for drills:

```bash
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ./scripts/restore-postgres.ps1 -BackupPath ./backups/bluecollarjob-YYYYMMDD-HHMMSS.dump
```

## 12. Known Staging Limitations

- Meta WhatsApp sending is available behind `WHATSAPP_PROVIDER=meta`; approved templates, opt-in review, and real credentials are still required before live traffic.
- Real Aadhaar/e-KYC is not enabled.
- Real UPI/payout is not enabled.
- Linode Object Storage is scaffolded but not wired to real credentials yet.
- Move the project outside OneDrive before regenerating `frontend/package-lock.json` and switching CI/Docker to `npm ci`.
