# PostgreSQL Setup Guide

**For dapp-server Project**
**Date:** January 21, 2026

This guide provides step-by-step instructions to install and configure PostgreSQL for the dapp-server project on macOS and Linux.

---

## Table of Contents

1. [macOS Setup](#macos-setup)
2. [Linux Setup (Ubuntu/Debian)](#linux-setup-ubuntudebian)
3. [Linux Setup (CentOS/RHEL/Fedora)](#linux-setup-centosrhelfedora)
4. [Database Configuration](#database-configuration)
5. [Application Setup](#application-setup)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)

---

## macOS Setup

### Step 1: Install PostgreSQL via Homebrew

```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install PostgreSQL
brew install postgresql@14

# Or for the latest version
brew install postgresql
```

### Step 2: Start PostgreSQL Service

```bash
# Start PostgreSQL service
brew services start postgresql@14

# Or for latest version
brew services start postgresql

# Verify service is running
brew services list | grep postgresql
```

**Expected output:**
```
postgresql@14 started allen ~/Library/LaunchAgents/homebrew.mxcl.postgresql@14.plist
```

### Step 3: Test PostgreSQL Connection

```bash
# Test connection (default user is your macOS username)
psql -d postgres -c "SELECT version();"
```

### Step 4: Create Database

```bash
# Create the dapp_server database
psql -d postgres -c "CREATE DATABASE dapp_server;"

# Verify database was created
psql -d postgres -c "\l" | grep dapp_server
```

**Expected output:**
```
 dapp_server | allen | UTF8 | en_US.UTF-8 | en_US.UTF-8 |
```

### Step 5: (Optional) Create Dedicated User

```bash
# Create a dedicated user for the application
psql -d postgres << EOF
CREATE USER dapp_user WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;
ALTER DATABASE dapp_server OWNER TO dapp_user;
EOF

# Test connection with new user
psql -U dapp_user -d dapp_server -c "SELECT 1;"
```

---

## Linux Setup (Ubuntu/Debian)

### Step 1: Install PostgreSQL

```bash
# Update package list
sudo apt update

# Install PostgreSQL
sudo apt install postgresql postgresql-contrib

# Check PostgreSQL version
psql --version
```

### Step 2: Start PostgreSQL Service

```bash
# Start PostgreSQL service
sudo systemctl start postgresql

# Enable auto-start on boot
sudo systemctl enable postgresql

# Check service status
sudo systemctl status postgresql
```

**Expected output:**
```
● postgresql.service - PostgreSQL RDBMS
   Loaded: loaded
   Active: active (running)
```

### Step 3: Access PostgreSQL

```bash
# Switch to postgres user
sudo -i -u postgres

# Access PostgreSQL prompt
psql
```

### Step 4: Create Database and User

**Inside the PostgreSQL prompt:**

```sql
-- Create database
CREATE DATABASE dapp_server;

-- Create user with password
CREATE USER dapp_user WITH PASSWORD 'your_secure_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;

-- Make user the owner
ALTER DATABASE dapp_server OWNER TO dapp_user;

-- Exit PostgreSQL prompt
\q
```

**Exit from postgres user:**

```bash
exit
```

### Step 5: Configure PostgreSQL Authentication (if needed)

```bash
# Edit pg_hba.conf to allow password authentication
sudo nano /etc/postgresql/14/main/pg_hba.conf
```

**Add or modify this line:**
```
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local   dapp_server     dapp_user                               md5
host    dapp_server     dapp_user       127.0.0.1/32            md5
```

**Restart PostgreSQL:**
```bash
sudo systemctl restart postgresql
```

### Step 6: Test Connection

```bash
# Test connection with the new user
psql -U dapp_user -d dapp_server -h localhost -c "SELECT version();"
```

---

## Linux Setup (CentOS/RHEL/Fedora)

### Step 1: Install PostgreSQL

```bash
# CentOS/RHEL 8
sudo dnf install postgresql-server postgresql-contrib

# Fedora
sudo dnf install postgresql-server postgresql-contrib

# Initialize database cluster
sudo postgresql-setup --initdb
```

### Step 2: Start PostgreSQL Service

```bash
# Start PostgreSQL
sudo systemctl start postgresql

# Enable auto-start
sudo systemctl enable postgresql

# Check status
sudo systemctl status postgresql
```

### Step 3: Create Database and User

```bash
# Switch to postgres user
sudo -u postgres psql

# Inside psql prompt:
CREATE DATABASE dapp_server;
CREATE USER dapp_user WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;
ALTER DATABASE dapp_server OWNER TO dapp_user;
\q
```

### Step 4: Configure Authentication

```bash
# Edit pg_hba.conf
sudo vi /var/lib/pgsql/data/pg_hba.conf
```

**Add:**
```
local   dapp_server     dapp_user                               md5
host    dapp_server     dapp_user       127.0.0.1/32            md5
```

**Restart:**
```bash
sudo systemctl restart postgresql
```

---

## Database Configuration

### For macOS (using your username)

Create or edit `.config/.env`:

```bash
cd /Users/allen/Professional/ymca-wellness-cafe/dappServer
mkdir -p .config
nano .config/.env
```

**Add the following:**

```env
# Smart Contract Hashes (update with your actual values)
ADD_ACTIVITY_CONTRACT=your_activity_contract_hash_here
ADD_ADMIN_CONTRACT=your_admin_contract_hash_here
TRANSFER_CONTRACT=your_transfer_contract_hash_here

# File Paths (update with your actual paths)
ACTIVITY_UPDATE_PATH=/path/to/activity/updates.json
ADD_ADMIN_PATH=/path/to/admin/updates.json

# PostgreSQL Database Configuration (macOS default)
DB_HOST=localhost
DB_PORT=5432
DB_USER=allen
DB_PASSWORD=
DB_NAME=dapp_server
DB_SSL_MODE=disable
```

**Note:** On macOS with Homebrew PostgreSQL, the default user is your system username (allen) and typically has no password for local connections.

### For Linux (using dedicated user)

```bash
cd /path/to/dappServer
mkdir -p .config
nano .config/.env
```

**Add the following:**

```env
# Smart Contract Hashes (update with your actual values)
ADD_ACTIVITY_CONTRACT=your_activity_contract_hash_here
ADD_ADMIN_CONTRACT=your_admin_contract_hash_here
TRANSFER_CONTRACT=your_transfer_contract_hash_here

# File Paths (update with your actual paths)
ACTIVITY_UPDATE_PATH=/path/to/activity/updates.json
ADD_ADMIN_PATH=/path/to/admin/updates.json

# PostgreSQL Database Configuration (Linux with dedicated user)
DB_HOST=localhost
DB_PORT=5432
DB_USER=dapp_user
DB_PASSWORD=your_secure_password
DB_NAME=dapp_server
DB_SSL_MODE=disable

# For production, use:
# DB_SSL_MODE=require
```

---

## Application Setup

### Step 1: Copy Environment Template

```bash
# Copy the example file
cp .env.example .config/.env

# Edit with your values
nano .config/.env
```

### Step 2: Update Configuration Values

**Required fields to update:**

1. **Contract Hashes** - Get these from your deployed Rubix smart contracts
2. **File Paths** - Update with actual paths to your JSON files
3. **Database Credentials** - Update based on your PostgreSQL setup

### Step 3: Build the Application

```bash
# Build the dapp-server binary
go build -o dapp-server

# Verify build
./dapp-server --help
```

### Step 4: Run the Server

```bash
# Run the server
./dapp-server
```

**Expected output:**

```
Initializing PostgreSQL database...
✅ PostgreSQL database initialized successfully
📊 Connection pool: max_open=25, max_idle=5
🚀 Transfer queue worker started (sequential processing)
Server running on :9000
```

---

## Verification

### Step 1: Check Database Connection

**macOS:**
```bash
psql -d dapp_server -c "SELECT version();"
```

**Linux:**
```bash
psql -U dapp_user -d dapp_server -h localhost -c "SELECT version();"
```

### Step 2: Verify Tables Were Created

**macOS:**
```bash
psql -d dapp_server -c "\dt"
```

**Linux:**
```bash
psql -U dapp_user -d dapp_server -h localhost -c "\dt"
```

**Expected output:**
```
                  List of relations
 Schema |       Name       | Type  |  Owner
--------+------------------+-------+----------
 public | transfer_status  | table | allen (or dapp_user)
(1 row)
```

### Step 3: Check Table Schema

**macOS:**
```bash
psql -d dapp_server -c "\d transfer_status"
```

**Linux:**
```bash
psql -U dapp_user -d dapp_server -h localhost -c "\d transfer_status"
```

**Expected columns:**
- request_id (VARCHAR PRIMARY KEY)
- blockchain_tx_id (VARCHAR)
- block_id (VARCHAR)
- activity_ids (TEXT)
- user_did (VARCHAR)
- admin_did (VARCHAR)
- reward_points (INTEGER)
- status (VARCHAR)
- message (TEXT)
- contract_hash (VARCHAR)
- error_details (TEXT)
- ft_transfer_txid (VARCHAR)
- queued_at, started_at, completed_at (TIMESTAMP)
- created_at, updated_at (TIMESTAMP)

### Step 4: Check Indexes

**macOS:**
```bash
psql -d dapp_server -c "\di"
```

**Linux:**
```bash
psql -U dapp_user -d dapp_server -h localhost -c "\di"
```

**Expected indexes:**
- idx_blockchain_tx_id
- idx_block_id
- idx_status
- idx_queued_at
- idx_admin_did
- idx_ft_transfer_txid

### Step 5: Test Application Endpoint

```bash
# Test health check
curl http://localhost:9000/health

# Test transfer endpoint with sample data
curl -X POST http://localhost:9000/api/v1/transfer/reward \
  -H "Content-Type: application/json" \
  -d '{
    "user_did": "bafybmidihrblwdyauw7q7sw5b4dch77zjuiqqyqtwxcs2j3nnfftxjkbeq",
    "admin_did": "bafybmicjpbgevsegrj3hbdyeilprk5mdbtdrkiec7hgmi3kdrcfhexvk6q",
    "activity_id": ["activity_001"],
    "reward_points": 100
  }'
```

---

## Troubleshooting

### Issue 1: Connection Refused

**Error:**
```
Failed to initialize database: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution:**

**macOS:**
```bash
# Check if PostgreSQL is running
brew services list | grep postgresql

# Start PostgreSQL if not running
brew services start postgresql@14
```

**Linux:**
```bash
# Check status
sudo systemctl status postgresql

# Start if not running
sudo systemctl start postgresql
```

### Issue 2: Database Does Not Exist

**Error:**
```
Failed to initialize database: database "dapp_server" does not exist
```

**Solution:**

**macOS:**
```bash
psql -d postgres -c "CREATE DATABASE dapp_server;"
```

**Linux:**
```bash
sudo -u postgres psql -c "CREATE DATABASE dapp_server;"
```

### Issue 3: Authentication Failed

**Error:**
```
Failed to initialize database: password authentication failed for user "dapp_user"
```

**Solution:**

1. **Check password in `.config/.env`** - Ensure it matches the password set in PostgreSQL

2. **Reset password if needed:**

**macOS:**
```bash
psql -d postgres -c "ALTER USER dapp_user WITH PASSWORD 'new_password';"
```

**Linux:**
```bash
sudo -u postgres psql -c "ALTER USER dapp_user WITH PASSWORD 'new_password';"
```

3. **Update `.config/.env`** with the new password

### Issue 4: Role Does Not Exist (macOS)

**Error:**
```
FATAL: role "postgres" does not exist
```

**Solution:**

On macOS, the default user is your system username, not "postgres". Update `.config/.env`:

```env
DB_USER=allen  # Replace with your actual username
DB_PASSWORD=   # Usually empty for local connections
```

### Issue 5: Permission Denied

**Error:**
```
permission denied for database dapp_server
```

**Solution:**

**Grant proper permissions:**

**macOS:**
```bash
psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE dapp_server TO allen;"
```

**Linux:**
```bash
sudo -u postgres psql << EOF
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;
ALTER DATABASE dapp_server OWNER TO dapp_user;
EOF
```

### Issue 6: Cannot Connect to Server Socket

**Error:**
```
could not connect to server: No such file or directory
```

**Solution:**

PostgreSQL socket file location might be different. Check your PostgreSQL configuration:

```bash
# Find socket directory
psql -d postgres -c "SHOW unix_socket_directories;"

# Or check configuration file
cat /usr/local/var/postgresql@14/postgresql.conf | grep unix_socket
```

### Issue 7: Port Already in Use

**Error:**
```
bind: address already in use
```

**Solution:**

1. **Check if another PostgreSQL instance is running:**

```bash
# macOS
lsof -i :5432

# Linux
sudo netstat -tulpn | grep 5432
```

2. **Use a different port** by updating `postgresql.conf` and `.config/.env`

### Issue 8: Connection Pool Exhausted

**Error:**
```
too many clients already
```

**Solution:**

1. **Increase PostgreSQL max_connections** in `postgresql.conf`:

```
max_connections = 100
```

2. **Adjust application pool settings** in `database/db.go`:

```go
db.SetMaxOpenConns(50)  // Increase from 25
db.SetMaxIdleConns(10)  // Increase from 5
```

3. **Restart PostgreSQL:**

**macOS:**
```bash
brew services restart postgresql@14
```

**Linux:**
```bash
sudo systemctl restart postgresql
```

---

## Quick Reference Commands

### macOS Quick Setup

```bash
# Install
brew install postgresql@14

# Start service
brew services start postgresql@14

# Create database
psql -d postgres -c "CREATE DATABASE dapp_server;"

# Test connection
psql -d dapp_server -c "SELECT 1;"
```

### Linux Quick Setup

```bash
# Install (Ubuntu/Debian)
sudo apt install postgresql postgresql-contrib

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE dapp_server;
CREATE USER dapp_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE dapp_server TO dapp_user;
ALTER DATABASE dapp_server OWNER TO dapp_user;
EOF

# Test connection
psql -U dapp_user -d dapp_server -h localhost -c "SELECT 1;"
```

### Database Management

```bash
# List databases
psql -l

# Connect to database
psql -d dapp_server

# List tables
\dt

# Describe table
\d transfer_status

# List indexes
\di

# Check table size
SELECT pg_size_pretty(pg_total_relation_size('transfer_status'));

# Check active connections
SELECT count(*) FROM pg_stat_activity WHERE datname='dapp_server';

# View recent records
SELECT * FROM transfer_status ORDER BY created_at DESC LIMIT 10;
```

---

## Security Best Practices

### 1. Use Strong Passwords

```bash
# Generate a strong password
openssl rand -base64 32
```

### 2. Enable SSL for Production

Update `.config/.env`:
```env
DB_SSL_MODE=require
```

### 3. Restrict Network Access

Edit `pg_hba.conf` to allow only specific IPs:
```
host    dapp_server    dapp_user    10.0.0.0/8    md5
```

### 4. Regular Backups

```bash
# Backup database
pg_dump -U dapp_user dapp_server > backup_$(date +%Y%m%d).sql

# Restore database
psql -U dapp_user dapp_server < backup_20260121.sql
```

### 5. Never Commit Credentials

Ensure `.config/.env` is in `.gitignore`:
```bash
echo ".config/.env" >> .gitignore
```

---

## Additional Resources

- **PostgreSQL Official Documentation:** https://www.postgresql.org/docs/
- **Homebrew PostgreSQL Guide:** https://wiki.postgresql.org/wiki/Homebrew
- **PostgreSQL Performance Tuning:** https://wiki.postgresql.org/wiki/Performance_Optimization
- **pg_hba.conf Documentation:** https://www.postgresql.org/docs/current/auth-pg-hba-conf.html

---

## Summary

You now have PostgreSQL installed and configured for the dapp-server project. The application will automatically create the necessary tables and indexes on first run.

**Key Points:**
- ✅ macOS uses your system username as the default PostgreSQL user
- ✅ Linux typically requires creating a dedicated user
- ✅ Connection pooling is configured for 25 concurrent connections
- ✅ Database schema is created automatically by the application
- ✅ All environment variables are loaded from `.config/.env`

**Next Steps:**
1. Update `.config/.env` with your actual contract hashes and file paths
2. Run `./dapp-server` to start the application
3. Test the API endpoints
4. Monitor PostgreSQL performance and adjust connection pool settings as needed
