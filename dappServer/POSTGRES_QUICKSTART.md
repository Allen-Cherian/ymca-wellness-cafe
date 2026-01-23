# PostgreSQL Quick Setup

**5-Minute Setup Guide for dapp-server**

---

## macOS

```bash
# 1. Install PostgreSQL (if not installed)
brew install postgresql@14

# 2. Start PostgreSQL
brew services start postgresql@14

# 3. Create database
psql -d postgres -c "CREATE DATABASE dapp_server;"

# 4. Verify
psql -d dapp_server -c "SELECT 1;"
```

**Configure `.config/.env`:**
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=allen
DB_PASSWORD=
DB_NAME=dapp_server
DB_SSL_MODE=disable
```

---

## Linux (Ubuntu/Debian)

```bash
# 1. Install PostgreSQL
sudo apt update && sudo apt install -y postgresql postgresql-contrib

# 2. Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 3. Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE dapp_server;
CREATE USER dapp_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;
ALTER DATABASE dapp_server OWNER TO dapp_user;
EOF

# 4. Verify
psql -U dapp_user -d dapp_server -h localhost -W -c "SELECT 1;"
```

**Configure `.config/.env`:**
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=dapp_user
DB_PASSWORD=your_password
DB_NAME=dapp_server
DB_SSL_MODE=disable
```

---

## Run Application

```bash
# Start the server (tables created automatically)
./dapp-server
```

**Expected Output:**
```
✅ PostgreSQL database initialized successfully
📊 Connection pool: max_open=25, max_idle=5
Server running on :9000
```

---

## Troubleshooting

| Error | Solution |
|-------|----------|
| Connection refused | `brew services start postgresql@14` (macOS) or `sudo systemctl start postgresql` (Linux) |
| Database does not exist | Run step 3 again |
| Authentication failed | Check password in `.config/.env` |
| Role "postgres" does not exist (macOS) | Use your username instead: `DB_USER=allen` |

---

## Done!

The application will automatically create all tables, indexes, and schema on first run.
