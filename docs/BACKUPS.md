# Database Backup & Restore Guide

This guide covers the procedures for backing up and restoring the GoPickup PostgreSQL database in production.

## 1. Production Backup Strategy

We recommend a **Daily Automated Backup** retained for 30 days, plus **WAL Archiving** (Point-in-Time Recovery) if supported by your managed provider (AWS RDS, Google Cloud SQL, DigitalOcean).

### Backup Contents
- **Primary**: The PostgreSQL database (`gopickup_prod`).
- **Secondary**: Any object storage buckets (S3/GCS) used for profile pictures/documents.

## 2. Manual Backup Command (pg_dump)

To create a manual backup of the database:

```bash
# Set variables
DB_HOST=your-db-host
DB_USER=postgres
DB_NAME=gopickup_prod
DATE=$(date +%Y-%m-%d_%H-%M-%S)

# Run pg_dump (prompts for password)
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME -F c -b -v -f "gopickup_backup_$DATE.dump"

# Verify file size
ls -lh "gopickup_backup_$DATE.dump"
```

*Note: `-F c` creates a custom format archive, which is compressed and allows selective restore.*

### Using Docker (if DB is containerized)
```bash
docker exec -t gopickup-db pg_dump -U postgres -d gopickup_prod -F c > dump_$(date +%F).dump
```

## 3. Automated Backup (Cron Example)

For a self-hosted VPS, use a cron job to dump and upload to S3.

1.  Create a script `backup.sh`:
    ```bash
    #!/bin/bash
    TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
    FILENAME="/backups/gopickup_$TIMESTAMP.dump"
    
    # Dump
    PGPASSWORD=your_password pg_dump -h localhost -U postgres -d gopickup_prod -F c -f $FILENAME
    
    # Upload to S3 (requires aws-cli)
    aws s3 cp $FILENAME s3://your-backup-bucket/db/
    
    # Cleanup local (keep last 7 days)
    find /backups -name "*.dump" -mtime +7 -delete
    ```

2.  Add to crontab (`crontab -e`):
    ```
    0 2 * * * /path/to/backup.sh >> /var/log/db_backup.log 2>&1
    ```

## 4. Restore Guide

**WARNING**: Restoring overwrites existing data. Ensure you are targeting the correct database.

### Restore to a clean database
```bash
# 1. Create fresh DB
createdb -h $DB_HOST -U $DB_USER gopickup_restore_test

# 2. Restore
pg_restore -h $DB_HOST -U $DB_USER -d gopickup_restore_test -v "gopickup_backup_2023-XX-XX.dump"
```

### Restore to Production (DANGER)
1.  Stop the application to prevent writes.
2.  Rename the current DB (backup safety):
    ```sql
    ALTER DATABASE gopickup_prod RENAME TO gopickup_prod_old;
    CREATE DATABASE gopickup_prod;
    ```
3.  Run restore:
    ```bash
    pg_restore -h $DB_HOST -U $DB_USER -d gopickup_prod -v "backup_file.dump"
    ```
4.  Start the application.
