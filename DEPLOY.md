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

---


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
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
