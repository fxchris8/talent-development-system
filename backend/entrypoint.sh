#!/bin/sh
set -e

echo "========================================="
echo "TDS Backend - Starting entrypoint script"
echo "========================================="

# Wait for MySQL to be ready
echo "Waiting for MySQL to be ready..."
until nc -z -v -w30 $DB_HOST $DB_PORT
do
  echo "Waiting for database connection at $DB_HOST:$DB_PORT..."
  sleep 2
done

echo "✅ MySQL is ready!"

# Run migrations
echo "Running database migrations..."
# A failed migration must stop the container: starting the API on a half-migrated schema
# hides the real error behind confusing runtime failures.
if ! ./migrate 2>&1; then
  echo "❌ Migration failed. The API is not started; fix the error above and restart."
  exit 1
fi
echo "✅ Migrations completed"

# Run seeder (optional, comment out if not needed every time)
echo "Running database seeder..."
if ./seed; then
  echo "✅ Seeding completed successfully"
else
  echo "⚠️  Seeding failed, but continuing..."
fi

# Start the API server
echo "Starting API server..."
exec ./api
