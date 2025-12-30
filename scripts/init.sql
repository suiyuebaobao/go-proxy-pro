-- Go-AIProxy Database Initialization Script
-- This script is automatically run by MySQL on first container start
-- via docker-entrypoint-initdb.d/

-- The Go application uses GORM AutoMigrate to create all tables:
-- - users (with default admin: admin / admin123)
-- - api_keys
-- - accounts
-- - account_groups
-- - account_group_members
-- - ai_models (with default models)
-- - packages
-- - user_packages
-- - proxies
-- - request_logs
-- - daily_usage
-- - usage_records
-- - operation_logs
-- - system_configs (with default configs)
-- - client_types
-- - client_filter_rules
-- - client_filter_config
-- - error_messages (with default messages)
-- - error_rules (with default rules)
-- - model_mappings

-- Set character set and collation for the database
ALTER DATABASE aiproxy CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
