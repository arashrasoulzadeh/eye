# قراردادهای اصلی API

## سلامت

GET /healthz بدون احراز هویت، پاسخ JSON برابر {"status":"ok"} دارد. روش دیگر ۴۰۵ است.

## نشست

POST /sessions با کلید مستأجر و بدنه شامل tenant_id الزامی، user_id و device اختیاری. پاسخ مدل نشست با فیلد ID است. GET با احراز هویت مدیر و session_id جزئیات یا بدون آن فهرست صفحه‌بندی‌شده می‌دهد. پارامترهای tenant_id، from_ts، to_ts، filter، limit، offset، sort، dir و include_user_tags قابل استفاده‌اند. زمان‌ها میلی‌ثانیه یونیکس هستند. PUT/PATCH با session_id و بدنه device، ip، user_agent اطلاعات نشست را تغییر می‌دهد.

## دریافت

POST /ingest یا /ingest/stream با کلید مستأجر و آرایه events. پاسخ accepted و rejected دارد؛ rejected شامل نمونه‌برداری حذف‌شده نیز هست. خطای ۴۲۹ نشان‌دهنده محدودیت نرخ است.

## پرس‌وجو

GET /query با احراز هویت مدیر. پارامترهای tenant_id، session_id، event_name، dims_contains، sort، from_ts، to_ts، limit، offset و cursor. پیش‌فرض limit برابر ۱۰۰ است. page=true پاسخ items، total، limit، offset می‌دهد؛ در غیر این صورت آرایه و در صورت وجود هدر X-Next-Cursor.

## SQL

GET /query-explorer?tenant_id=... کلیدهای ویژگی را برمی‌گرداند. POST با نشست مدیر و بدنه زیر اجرا می‌شود. پاسخ columns، rows، truncated و duration_ms دارد. بدنه حداکثر ۳۲۷۶۸ بایت است و فیلد ناشناخته رد می‌شود.

```json
{"tenant_id":"example","sql":"SELECT COUNT(*) FROM events","from_ts":0,"to_ts":0}
```
