# API تحلیل‌ها

همه مسیرهای زیر با احراز هویت مدیر ثبت شده‌اند. tenant_id را مشخص کنید. زمان‌ها میلی‌ثانیه یونیکس هستند.

| Route | HTTP | پارامتر و رفتار |
|---|---|---|
| `/analytics/funnel` | POST | بدنه JSON مستأجر، مراحل غیرخالی و زمان |
| `/analytics/funnel/pinned` | GET / POST / DELETE | خواندن، جایگزینی یا پاک‌کردن قیف؛ نام و دو مرحله متمایز برای POST |
| `/analytics/funnel/suggested` | GET | مستأجر و بازه زمانی |
| `/analytics/retention` | GET | دانه‌بندی روزانه پیش‌فرض یا هفتگی؛ بازه زمانی |
| `/analytics/views` | GET / POST | فهرست یا ذخیره مدل نما؛ مستأجر الزامی |
| `/analytics/session-events` | GET | رویدادها در هر نشست |
| `/analytics/devices` | GET | تفکیک دستگاه |
| `/analytics/top-n` | GET | نام رویداد، حد پیش‌فرض ۱۰ |
| `/analytics/sliding-window` | GET | نام رویداد الزامی؛ پنجره پیش‌فرض ۶۰ ثانیه |
| `/analytics/distinct-sessions` | GET | نام رویداد؛ تعداد نشست متمایز |
| `/analytics/percentiles` | GET | صدک رویدادهای نشست |
| `/analytics/geography` | GET | بعد پیش‌فرض کشور |
| `/analytics/buckets` | GET | نام رویداد، بازه پیش‌فرض ساعت، زمان آغاز و پایان صعودی الزامی |

نکته: هندلر views شاخه‌های PUT و DELETE با شناسه مسیر دارد، اما ثبت مسیر فعلی فقط /analytics/views است؛ دسترسی به زیرمسیر را تضمین نمی‌کنیم.

برای قیف نمونه:

```json
{"tenant_id":"example","steps":["page_view","purchase"],"from_ts":1700000000000,"to_ts":1700086400000}
```
