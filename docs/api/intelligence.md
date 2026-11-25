# Intelligence API contracts

Use `/intelligence/users` with admin authentication and the action query parameter. Except for the first four actions, tenant_id is required. Methods below are intended usage; some handler branches do not enforce the method.

| Action | HTTP | Contract |
|---|---|---|
| `model-status` | GET | Installed and latest release |
| `download-model` | POST | Install release |
| `training-csv` | GET | CSV; optional tenant_id, from_ts, to_ts |
| `inference-log` | GET | Latest 50 runs; optional tenant and time |
| `label` | POST | JSON fingerprint, label, source; 204 |
| `suggestions` | GET | Up to 50 suggestions |
| `review` | POST | JSON fingerprint, label, decision |
| `pipeline` | POST | Optional retry_id; pipeline run |
| `pipeline-status` | GET | Latest 20 runs |
| `readiness` | GET | label defaults to buyer |
| `training` | GET | label; latest 20 runs |
| `models` | GET | label; up to 100 models |
| `model-layers` | GET | label; up to 100 layers |
| `evaluations` | GET | label; up to 100 evaluations |
| `train` | POST | label defaults to auto; 422 on failure |
| `model` | GET | task defaults to buyer; scope and manifest |
| `evaluation` | GET / POST | Read or evaluate latest; label defaults to behavior |
| `quality` | GET / POST | Read or assess data quality |
| `alerts` | GET | status; up to 100 alerts |
| `resolve-alert` | POST | id required; 204 |
| `promote` | POST | label defaults to behavior |
| `rollback` | POST | label defaults to behavior |
| `export` | POST | label defaults to buyer; scope defaults to global |
| `refresh` | POST | Returns refreshed count |

Without action, the handler lists users with limit. Run model-changing actions only with appropriate authorization.
