# 🚀 Production Deployment Guide

## ⚠️ PERBEDAAN DEV vs PRODUCTION

| Aspek              | Development              | Production                  |
| ------------------ | ------------------------ | --------------------------- |
| **Dockerfile**     | Simple, untuk hot reload | Multi-stage, optimized      |
| **Source Code**    | Bind mount (live edit)   | Built into image            |
| **Environment**    | `NODE_ENV=development`   | `NODE_ENV=production`       |
| **Secrets**        | Hardcoded di compose     | `.env` atau secrets manager |
| **Restart Policy** | `unless-stopped`         | `always`                    |
| **Gin Mode**       | `debug`                  | `release`                   |
| **Volumes**        | Bind mount source        | Only persistent data        |
| **Build**          | `npm run dev`            | `npm run build` + optimize  |
| **Port**           | 3001, 8081               | 3000, 8080                  |
| **Hot Reload**     | ✅ Enabled               | ❌ Disabled                 |
| **Logging**        | Verbose                  | Structured                  |
| **Security**       | Relaxed                  | Hardened                    |

---

## 📋 Prerequisites

1. **Server dengan Docker & Docker Compose** (Ubuntu/Debian recommended)
2. **Domain name** (optional, tapi recommended)
3. **SSL Certificate** (untuk HTTPS)
4. **Reverse proxy** (Nginx/Traefik)

---

## 🛠️ Setup Production

### 1. Prepare Server

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Create app directory
sudo mkdir -p /opt/tds
cd /opt/tds
```

### 2. Clone Repository

```bash
# Clone dari Git (jangan include .env files!)
git clone <your-repo-url> .

# Atau upload via SCP/SFTP
```

### 3. Setup Environment Variables

```bash
# Copy dan edit environment file
cp .env.prod.example .env.prod

# Edit dengan credentials yang AMAN
nano .env.prod
```

**Variabel yang WAJIB diisi di .env.prod** (lihat contoh lengkap di `.env.prod.example`):

| Variabel | Untuk apa | Akibat bila kosong/salah |
| :-- | :-- | :-- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Koneksi MySQL eksternal | Backend menunggu database terus-menerus dan tidak pernah start |
| `JWT_SECRET` | Tanda tangan token login; dipakai backend DAN frontend (nilainya harus sama, sudah diatur compose) | Backend menolak start; semua user dilempar ke `/login` |
| `NEXT_PUBLIC_API_ENDPOINT` | URL backend yang dipanggil browser. **Ditanam saat build image frontend** | Frontend memanggil `localhost:8080` |
| `BACKEND_PUBLIC_URL` | URL backend untuk link file PPT/PDF materi yang disimpan ke database | Link materi mengarah ke `localhost` |
| `FRONTEND_URL` | Redirect setelah login SSO | Redirect ke `localhost:3000` |
| `SSO_*` | Login SSO (kosongkan semua bila tidak dipakai) | Tombol SSO tidak berfungsi |
| `OPENROUTER_API_KEY` | Generate materi PPT/PDF dan quiz | Fitur generate gagal |

Catatan:

- Login memakai cookie. Cookie hanya diberi tanda `Secure` bila request datang lewat HTTPS, jadi
  server yang diakses lewat `http://` tetap bisa login. Bila memakai reverse proxy HTTPS (nginx,
  Traefik), pastikan proxy mengirim header `X-Forwarded-Proto: https`.
- Frontend dan backend harus berada di domain induk yang sama (mis. `app.domain.com` dan
  `api.domain.com`, atau host yang sama dengan port berbeda) supaya cookie login ikut terkirim.
- Migrasi database berjalan otomatis setiap backend start (menambah tabel/kolom baru, tidak
  menghapus data). Rilis ini menambah kolom `training_status`, `exited_program`,
  `program_exited_at` pada tabel `reports`.
- API hanya menerima panggilan browser dari frontend: origin yang cocok dengan `FRONTEND_URL`
  (atau `CORS_ALLOWED_ORIGINS`), atau yang satu situs dengan API (host sama, atau domain induk
  sama). Bila frontend ditolak, log backend menulis `CORS: origin ... ditolak` beserta
  variabel yang harus diisi.
- Seeder berjalan setiap backend start dengan `SEED_ON_START=auto`: hanya tabel yang masih
  kosong yang diisi, jadi data yang sudah dihapus admin tidak muncul lagi. Pakai `always`
  untuk perilaku lama (membuat ulang semua baris seed yang hilang) atau `never` untuk
  mematikannya.
- Materi hasil generate disimpan di volume `backend_materi`, gambar soal di `backend_storage`.
  Jangan jalankan `docker compose down -v` di production: itu menghapus kedua volume tersebut.

**Membuat nilai rahasia:**

```bash
# Generate JWT secret yang kuat
openssl rand -base64 32

# Generate MySQL passwords
openssl rand -base64 24
```

### 4. Build dan Deploy

```bash
# Build images (WAJIB memakai --env-file: NEXT_PUBLIC_API_ENDPOINT ditanam saat build)
docker-compose -f docker-compose.prod.yml --env-file .env.prod build

# Start services
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Check logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 5. Verify Deployment

```bash
# Check all containers running
docker-compose -f docker-compose.prod.yml ps

# Test health check
curl http://localhost:8080/health

# Test frontend (harus 200; halaman lain akan redirect ke /login)
curl -I http://localhost:3000/login

# Pastikan migrasi & start backend bersih
docker-compose -f docker-compose.prod.yml logs tds_backend | grep -iE "migrat|fatal|error" | tail -20

# Layanan suara materi (harus {"status": "ok", ...})
docker exec tds_backend_prod curl -s http://tds_tts:5002/health
```

Catatan:

- **Migrasi gagal = backend tidak start.** `entrypoint.sh` menghentikan container bila `./migrate`
  gagal, supaya API tidak jalan di atas skema yang setengah jadi. Baca error migrasinya di log.
- **Suara materi (`tds_tts`).** Container ini membacakan materi training dalam Bahasa Indonesia
  (mesin Piper, offline, tanpa API key). Model suara (~60 MB) diunduh saat build, jadi server
  butuh internet saat `build` saja. Audio dibuat sekali per materi (sekitar 1 menit untuk
  materi 15-20 menit), lalu disimpan di volume `backend_materi` sebagai `...-narasi.mp3`.
- **Login dibatasi.** 5 password salah dalam 10 menit mengunci pasangan IP + username selama
  10 menit (disimpan di memori, reset saat backend restart). Bila backend berada di belakang
  reverse proxy, pastikan proxy meneruskan IP asli klien (`X-Forwarded-For`).

---

## 🔒 Security Hardening

### 1. Environment Variables

```bash
# Jangan gunakan default passwords!
# Gunakan secrets manager untuk production serius

# Option 1: Docker Secrets (Docker Swarm)
# Option 2: Environment dari CI/CD (GitHub Actions, GitLab CI)
# Option 3: HashiCorp Vault
# Option 4: AWS Secrets Manager / Azure Key Vault
```

### 2. Firewall

```bash
# Install UFW
sudo apt install ufw -y

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS (jika pakai reverse proxy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# JANGAN expose port 3000, 8080 ke public!
# Gunakan reverse proxy (Nginx/Traefik)

# Enable firewall
sudo ufw enable
```

### 3. Nginx Reverse Proxy (Recommended)

```nginx
# /etc/nginx/sites-available/tds

# Backend API
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Frontend
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/tds /etc/nginx/sites-enabled/

# Test config
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

### 4. SSL dengan Let's Encrypt

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx -y

# Generate certificates
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com -d api.yourdomain.com

# Auto-renewal sudah dihandle oleh certbot
```

---

## 🔄 Update & Maintenance

### Update Application

```bash
cd /opt/tds

# Pull latest code
git pull origin main

# Rebuild images
docker-compose -f docker-compose.prod.yml build

# Restart with zero downtime (gunakan healthchecks)
docker-compose -f docker-compose.prod.yml up -d --no-deps --build tds_backend
docker-compose -f docker-compose.prod.yml up -d --no-deps --build tds_frontend
```

### Backup Database

```bash
# Backup script
#!/bin/bash
BACKUP_DIR="/opt/backups/tds"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

docker exec mysql-container-prod mysqldump \
  -u tds_user_prod \
  -p'your-password' \
  tds > $BACKUP_DIR/tds_backup_$DATE.sql

# Compress
gzip $BACKUP_DIR/tds_backup_$DATE.sql

# Delete old backups (keep last 7 days)
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete
```

### Restore Database

```bash
# Unzip backup
gunzip tds_backup_20250102_120000.sql.gz

# Restore
docker exec -i mysql-container-prod mysql \
  -u tds_user_prod \
  -p'your-password' \
  tds < tds_backup_20250102_120000.sql
```

---

## 📊 Monitoring & Logs

### View Logs

```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f tds_backend

# Last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100 tds_backend
```

### Log Rotation

```bash
# Edit /etc/docker/daemon.json
sudo nano /etc/docker/daemon.json
```

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

```bash
# Restart Docker
sudo systemctl restart docker
```

### Monitoring Tools (Optional)

1. **Portainer** - Docker GUI management
2. **Prometheus + Grafana** - Metrics & dashboards
3. **ELK Stack** - Log aggregation
4. **Uptime Kuma** - Uptime monitoring

---

## 🐳 Docker Compose Production Commands

```bash
# Start (with env file)
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Stop
docker-compose -f docker-compose.prod.yml down

# Restart specific service
docker-compose -f docker-compose.prod.yml restart tds_backend

# View logs
docker-compose -f docker-compose.prod.yml logs -f

# Check status
docker-compose -f docker-compose.prod.yml ps

# Remove everything (⚠️ including volumes!)
docker-compose -f docker-compose.prod.yml down -v
```

---

## 🚀 CI/CD Pipeline (Optional)

### GitHub Actions Example

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Deploy to server
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.SERVER_HOST }}
          username: ${{ secrets.SERVER_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            cd /opt/tds
            git pull origin main
            docker-compose -f docker-compose.prod.yml build
            docker-compose -f docker-compose.prod.yml up -d
```

---

## ⚡ Performance Tips

### 1. Database Optimization

```sql
-- Add indexes untuk query yang sering
CREATE INDEX idx_user_email ON users(email);
CREATE INDEX idx_created_at ON reports(created_at);
```

### 2. Backend Optimization

```bash
# Gunakan connection pooling (sudah ada di GORM)
# Set proper GOMAXPROCS
ENV GOMAXPROCS=4
```

### 3. Frontend Optimization

- ✅ Sudah menggunakan Next.js production build
- ✅ Static assets di-optimize otomatis
- ✅ Image optimization dengan Next.js Image component

### 4. Database Connection Limits

```yaml
# di docker-compose.prod.yml MySQL
command:
  - --default-authentication-plugin=mysql_native_password
  - --max-connections=200
  - --innodb-buffer-pool-size=512M
```

---

## 📱 Health Checks & Alerts

### Setup Alert (Slack/Telegram)

```bash
# Script untuk monitoring
#!/bin/bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)

if [ $STATUS -ne 200 ]; then
  # Send alert
  curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"⚠️ TDS Backend is DOWN!"}' \
    YOUR_SLACK_WEBHOOK_URL
fi
```

### Cron Job

```bash
# Add to crontab
crontab -e

# Check every 5 minutes
*/5 * * * * /opt/tds/scripts/health-check.sh
```

---

## 🆘 Troubleshooting Production

### Container tidak start?

```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs tds_backend

# Check disk space
df -h

# Check memory
free -h

# Check Docker daemon
sudo systemctl status docker
```

### Database connection error?

```bash
# Check MySQL logs
docker logs mysql-container-prod

# Test connection dari backend container
docker exec tds_backend_prod nc -zv mysql-container-prod 3306

# Check environment variables
docker exec tds_backend_prod env | grep DB_
```

### High memory usage?

```bash
# Limit resources di docker-compose.prod.yml
services:
  tds_backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
```

---

## 📝 Checklist Deployment

- [ ] Environment variables di-set dengan aman
- [ ] Passwords di-ganti dari default
- [ ] JWT secret di-generate dengan secure
- [ ] Firewall dikonfigurasi
- [ ] Reverse proxy (Nginx) disetup
- [ ] SSL certificate di-install
- [ ] Database backup otomatis disetup
- [ ] Log rotation dikonfigurasi
- [ ] Health checks berfungsi
- [ ] Monitoring disetup
- [ ] Domain DNS pointing ke server
- [ ] `.env.prod` TIDAK di-commit ke Git

---

## 🔗 Additional Resources

- [Docker Production Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [12-Factor App Methodology](https://12factor.net/)
- [OWASP Security Guidelines](https://owasp.org/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)

---

**Deployment sukses? Jangan lupa monitoring! 🚀**
