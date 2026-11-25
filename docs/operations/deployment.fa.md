# استقرار عملیاتی

[راهنمای عملیاتی اصلی](https://github.com/arashrasoulzadeh/eye/blob/main/deploy/PRODUCTION.md) مرجع اجرای استقرار است.

فایل .env.example را به .env کپی کنید، مقادیر نمونه را جایگزین کنید و اسرار را خارج از Git نگه دارید. ADMIN_KEY، API_KEY_SALT، DATABASE_URL و CORS_ORIGINS دقیق را تنظیم کنید. حالت عملیاتی اعتبارنامه توسعه را رد می‌کند.

```sh
./deploy/deploy.sh
```

اسکریپت تصاویر آماده GHCR را دریافت می‌کند. برای استقرار تکرارپذیر نسخه تصاویر را ثابت کنید. برای PostgreSQL مقدار DB_DRIVER=postgres لازم است. Redis Streams با docker-compose.redis.yml فعال می‌شود. فایل‌های Kubernetes، Helm و Terraform در deploy/ هستند.

## مهاجرت و بازیابی

قبل از مهاجرت SQLite دریافت رویداد را متوقف کنید. پشتیبان بگیرید و حجم اصلی را حفظ کنید.

```sh
CONFIRM_MIGRATION=YES ./deploy/migrate.sh
```

اسکریپت deploy/backup/postgres-backup.sh را روزانه و restore-drill.sh را ماهانه روی مقصد مستقل اجرا کنید. پشتیبان و چک‌سام را جداگانه ذخیره کنید.

## امنیت و بررسی

از نقش غیرمالک پایگاه داده استفاده کنید. RLS را فقط پس از موفقیت آزمون‌های زمینه مستأجر فعال کنید. پس از استقرار سلامت، ایجاد نشست، دریافت رویداد و شاخص‌ها را بررسی کنید. قواعد Prometheus و گیرنده‌های Alertmanager را تنظیم کنید.
