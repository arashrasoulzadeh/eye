# مرجع SQL

کاوشگر گویش محدود SQL را روی رویدادهای مستأجر منتخب اجرا می‌کند، نه SQL دلخواه پایگاه داده.

## عبارت‌ها

توابع مجاز: `COUNT, SUM, AVG, MIN, MAX, ROUND, LOWER, UPPER, COALESCE, LENGTH, ABS`.

فیلدها: `event_id, tenant_id, session_id, user_id, type, event_name, timestamp, client_ts, ingest_ts, payload, dims, source, is_aggregated, aggregation_id, experiment_id, variant`.

برای رشته‌ها از نقل‌قول تکی استفاده کنید و برای درج آن در رشته، آن را دو بار بنویسید. نام مستعار را با AS مشخص کنید. ویژگی‌ها با `payload.country` یا `dims.country` قابل ارجاع هستند؛ ویژگی بدون پیشوند ابتدا در payload و سپس در dims جستجو می‌شود.

```sql
SELECT dims.country, COUNT(*) AS total FROM events
WHERE type = 'click' GROUP BY dims.country ORDER BY total DESC;
```

کامپایلر عبارت‌های WHERE، JOIN، INNER، LEFT، ON، EXISTS، AND، OR، NOT، IS NULL، IN، BETWEEN، LIKE، GROUP BY، HAVING، ORDER BY، LIMIT، OFFSET و DISTINCT را می‌شناسد. همه ترکیب‌ها الزاماً در هر دو پایگاه داده معتبر نیستند. UNION، INTERSECT، EXCEPT، WITH، INTO، OVER و RETURNING رد می‌شوند.

## محدودیت‌ها

- ۱۶۰۰۰ بایت ورودی و ۵۱۲ توکن.
- مهلت اجرای پنج ثانیه.
- حداکثر ۱۰۰۰ ردیف خروجی؛ پرچم truncated را بررسی کنید.
- کشف ویژگی‌ها از ۲۰۰ رویداد اخیر و حداکثر ۵۰۰ کلید.
- انتخاب مستأجر الزامی است؛ بازه زمانی اختیاری بر زمان ثبت سمت کلاینت اعمال می‌شود.

[کد کامپایلر](https://github.com/arashrasoulzadeh/eye/blob/main/internal/service/query_explorer.go)
