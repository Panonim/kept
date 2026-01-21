# Deployment Guide for Kept

### Pull example .env file

```sh
wget https://raw.githubusercontent.com/Panonim/kept/refs/heads/main/.env.example -o .env
```

## Deploying with Docker Compose

```yaml
services:
  kept:
    image: ghcr.io/panonim/kept:latest
    container_name: kept
    environment:
      - PORT=3000
      - JWT_SECRET=your_jwt_secret
      - DB_ENCRYPTION_KEY=your_db_encryption_key
      - JWT_REFRESH_SECRET=your_refresh_secret
      - ALLOWED_ORIGINS=https://yourdomain.com
      - DISABLE_REGISTRATION=false
      - RUN_MIGRATIONS=false
      # App URL for email links
      - APP_URL=https://yourdomain.com
    volumes:
      - data:/data
    ports:
      - "80:80"     # Frontend
      - "3000:3000" # Backend
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  data:
```

- **Note:** Set all secrets (e.g., `JWT_SECRET`, `DB_ENCRYPTION_KEY`) securely, preferably using environment variables or Docker secrets.
- The SQLite database is stored in the `kept-data` volume. The database file is encrypted if `DB_ENCRYPTION_KEY` is set.
- **SMTP Configuration:** Email reminders are optional. If SMTP variables are not set, only push notifications will be sent.

---

## Email Reminder Setup (Optional)

Kept can send email reminders in addition to push notifications. Configure these environment variables:

```bash
SMTP_HOST=smtp.gmail.com        # Your SMTP server
SMTP_PORT=587                   # Usually 587 for TLS, 465 for SSL
SMTP_USER=your-email@gmail.com  # SMTP username
SMTP_PASS=your-app-password     # SMTP password or app password
SMTP_FROM=noreply@yourdomain.com # From address
SMTP_USE_TLS=true              # Use TLS (recommended)
APP_URL=https://yourdomain.com  # Your app URL for email links
```

### Common SMTP Providers:

**Gmail:**
- Host: `smtp.gmail.com`, Port: `587`
- Enable 2FA and create an [App Password](https://support.google.com/accounts/answer/185833)

**Outlook/Office365:**
- Host: `smtp-mail.outlook.com`, Port: `587`
- Use your account credentials

**SendGrid:**
- Host: `smtp.sendgrid.net`, Port: `587`
- Username: `apikey`, Password: Your API key

**Mailgun:**
- Host: `smtp.mailgun.org`, Port: `587`
- Use your Mailgun credentials

### Testing Email Configuration:

Once configured, test email sending via:
```bash
# Login to your Kept instance
curl -X POST http://localhost:3000/api/email/test \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Rate limit: Once per 10 minutes per server.

---

## Generating VAPID keys

For web-push notifications (required for background/persisted push), generate VAPID keys and set them as environment variables.

- Generate keys locally (Node.js required):

```sh
npx web-push generate-vapid-keys --json
```

Copy the resulting `publicKey` and `privateKey` into your environment.

- Example environment variables (Docker Compose `environment` block or systemd exports):

```
VAPID_PUBLIC_KEY=your-vapid-public-key-here
VAPID_PRIVATE_KEY=your-vapid-private-key-here
VAPID_SUBJECT=mailto:admin@yourdomain.com
```

- For Docker users: add these to the `kept` service `environment` in `docker-compose.yml`, or provide via Docker secrets for improved security.

## Bare Metal Deployment (No Docker)

To deploy Kept directly on a Linux server (without Docker), follow these steps:

1. **Install Dependencies:**
  - Go (>=1.21) for backend
  - Node.js (>=20) and npm for frontend
  - SQLite3 and SQLCipher (for encrypted database support)
  - nginx (or another web server) for serving the frontend

2. **Build the Backend:**
  ```sh
  cd backend
  go build -o kept-server
  ```

3. **Build the Frontend:**
  ```sh
  cd ../frontend
  npm install
  npm run build
  ```

4. **Configure Environment Variables:**
  - Set all required environment variables (e.g., `PORT`, `JWT_SECRET`, `DB_ENCRYPTION_KEY`, etc.) in your shell or a systemd service file.
  - Ensure the backend has write access to the data directory (e.g., `/var/lib/kept/data`).

5. **Run the Backend:**
  ```sh
  ./backend/kept-server
  ```

6. **Serve the Frontend:**
  - Copy the contents of `frontend/dist` to your web server's root (e.g., `/var/www/kept`).
  - Configure nginx or another web server to serve these static files on port 80 or your desired port.

7. **Database:**
  - The backend will create and migrate the SQLite database automatically in the configured data directory.
  - For encryption, set `DB_ENCRYPTION_KEY` before starting the backend. If the database already exists, this key is required to open it.

**Example systemd service for backend:**
```ini
[Unit]
Description=Kept Backend Service
After=network.target

[Service]
Type=simple
User=kept
WorkingDirectory=/opt/kept
ExecStart=/opt/kept/kept-server
Environment=PORT=3000
Environment=JWT_SECRET=your_jwt_secret
Environment=DB_ENCRYPTION_KEY=your_db_encryption_key
Environment=JWT_REFRESH_SECRET=your_refresh_secret
Environment=ALLOWED_ORIGINS=https://yourdomain.com
Environment=SMTP_HOST=smtp.gmail.com
Environment=SMTP_PORT=587
Environment=SMTP_USER=your-email@gmail.com
Environment=SMTP_PASS=your-app-password
Environment=SMTP_FROM=noreply@yourdomain.com
Environment=APP_URL=https://yourdomain.com
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
