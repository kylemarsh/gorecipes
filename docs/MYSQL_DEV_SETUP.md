# MySQL Development Server Setup

Instructions for setting up a local MySQL development server on Mac, loading the production dump, and applying migrations.

## Step 1: Install and Start MySQL

### Option A: Using Homebrew (Recommended)

```bash
# Install MySQL if not already installed
brew install mysql

# Start MySQL service
brew services start mysql

# Verify it's running
brew services list | grep mysql
```

### Option B: Using existing MySQL installation

If you already have MySQL installed:

```bash
# Start MySQL (if using Homebrew)
brew services start mysql

# OR start manually
mysql.server start
```

## Step 2: Create the Development Database

```bash
# Connect to MySQL as root (no password by default on fresh install)
mysql -u root

# At the MySQL prompt, create the database
CREATE DATABASE eats_test;

# Verify it was created
SHOW DATABASES;

# Exit MySQL
EXIT;
```

## Step 3: Load the Production Dump

```bash
# Load the dump into the eats_test database
mysql -u root eats_test < backups/dashery_eats_prod_dump_2026-03-20.sql

# Verify the data loaded correctly
mysql -u root eats_test -e "SELECT COUNT(*) as recipe_count FROM recipe;"
mysql -u root eats_test -e "SELECT COUNT(*) as label_count FROM label;"
```

Expected output:
- recipe_count: 492
- label_count: 42

## Step 4: Apply the Migration Script

```bash
# Apply the migration
mysql -u root eats_test < scripts/migration_label_standardization_20260320.sql

# Verify the migration worked
mysql -u root eats_test -e "SELECT label_id, label, icon, type FROM label WHERE label_id IN (24, 53, 54, 55) ORDER BY label_id;"
```

## Step 5: Test with the Go Server

```bash
# Build and run the server
go build
./gorecipes --config dev_mysql.config --debug

# In another terminal, test the API
curl http://localhost:8080/labels/
```

## Step 6: Create a Fresh Dump (Optional)

After migration, create a new dump for future reference:

```bash
# Dump the migrated database
mysqldump -u root eats_test > backups/dashery_eats_dev_post_migration_$(date +%Y-%m-%d).sql
```

## Troubleshooting

### MySQL won't start
```bash
# Check if MySQL is already running
ps aux | grep mysql

# If stuck, try stopping and restarting
brew services stop mysql
brew services start mysql
```

### Connection errors
If you get "Access denied for user 'root'", you may need to set a password or use a different user.

Check your MySQL connection settings:
```bash
mysql -u root
# If this fails, try:
mysql -u root -p
# (enter your password when prompted)
```

Then update `dev_mysql.config` DbDSN to match your setup:
- With password: `"DbDSN": "root:yourpassword@tcp(localhost)/eats_test"`
- Different user: `"DbDSN": "username:password@tcp(localhost)/eats_test"`

### Database already exists
If you need to start fresh:
```bash
mysql -u root -e "DROP DATABASE IF EXISTS eats_test; CREATE DATABASE eats_test;"
```

## Cleanup

When done with development:

```bash
# Stop the MySQL service
brew services stop mysql

# OR if started manually
mysql.server stop
```
