# Configuration reference

Configuration struct keys and source defaults. A struct key does not guarantee direct environment binding. Never commit secrets.

| Key | Source default |
|---|---|
| `port` | `8080` |
| `db_path` | `"eye.db"` |
| `db_driver` | `"sqlite"` |
| `database_url` | — |
| `cors_origins` | — |
| `api_key_ttl_hours` | — |
| `broker_url` | — |
| `broker_stream` | `"eye-events"` |
| `broker_group` | `"eye-consumers"` |
| `broker_consumer` | `"eye-server"` |
| `enable_postgres_rls` | — |
| `admin_key` | `"dev-insecure-admin-key"` |
| `api_key_salt` | `"dev-insecure-salt"` |
| `retention_hours` | `24*30` |
| `snapshot_dir` | `"./snapshots"` |
| `pii_fields` | — |
| `pii_detection` | — |
| `schema_mode` | — |
| `dead_letter_notify_url` | — |
| `smtp_host` | — |
| `smtp_port` | — |
| `smtp_from` | — |
| `smtp_user` | — |
| `smtp_password` | — |
| `otlp_exporter_endpoint` | — |
| `geoip_database_path` | `""` |
| `model_registry_dir` | `"./models"` |

ADMIN_KEY, API_KEY_SALT and DATABASE_URL support the _FILE suffix.
