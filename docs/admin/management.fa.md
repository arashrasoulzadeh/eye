# راهنمای مدیریت و عملیات

## هشدارها

`/alerts`

مستأجر، رویداد، نوع قانون، پنجره زمانی بر حسب ثانیه، آستانه شمارش، کلید بعد اختیاری و مقصد را تعیین کنید. قانون را بسازید، ویرایش، فعال، غیرفعال یا حذف کنید. پیش از فعال‌سازی مقصد وب‌هوک را بررسی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/alerts/page.tsx)

## گروه کاربران

`/cohorts`

مستأجر، نام، رویداد و حداقل شمارش را انتخاب کنید. گروه را بسازید، عضویت را محاسبه و نتیجه را بررسی کنید؛ گروه قدیمی را حذف کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/cohorts/page.tsx)

## آزمایش‌ها

`/experiments`

مستأجر، نام، گونه‌های جداشده با ویرگول و سهم ترافیک را تعیین کنید. آزمایش را بسازید یا ویرایش کنید. برای تخصیص گونه از شناسه آزمایش و نشست استفاده کنید. تخصیص به معنای معناداری آماری نیست.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/experiments/page.tsx)

## پرچم قابلیت

`/feature-flags`

مستأجر و کلید پرچم را انتخاب و وضعیت فعال بودن را ایجاد یا به‌روزرسانی کنید. پیش از حذف پرچم قدیمی مصرف‌کنندگان آن را بررسی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/feature-flags/page.tsx)

## تجمیع‌ها

`/aggregations`

مستأجر، نام، قانون JSON، رویداد خروجی، آستانه و پنجره را تعیین کنید. تعریف را بسازید یا ویرایش کنید و نتایج را تازه‌سازی یا برای گذشته محاسبه کنید. رویداد مشتق‌شده را از خام جدا کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/aggregations/page.tsx)

## عملیات

`/operations`

پرچم‌ها، تجمیع‌ها، ناهنجاری‌ها، حسابرسی و ارسال‌های ناموفق را بررسی کنید. پیش از تازه‌سازی تجمیع یا تغییر پرچم، محدوده مستأجر و رکوردهای اخیر را بررسی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/operations/page.tsx)

## فشرده‌سازی

`/compression`

موجودی ذخیره‌سازی، نوع رویداد، گروه‌های سنی، شاخص‌ها و صرفه‌جویی تخمینی را بررسی کنید. حذف داده عملیاتی مخرب است؛ ابتدا نیازهای نگهداری و پشتیبان را بررسی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/compression/page.tsx)

## ارسال‌های ناموفق

`/dead-letters`

ارسال ناموفق را بر اساس نشانی، شناسه یا خطا فیلتر کنید و محتوا، تعداد تلاش و آخرین خطا را بررسی کنید. پیش از بازپخش مقصد را اصلاح کنید؛ فقط موارد غیرضروری را حذف کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/dead-letters/page.tsx)

## طرح‌ها

`/schemas`

مستأجر، نام طرح و تعریف JSON را انتخاب کنید. طرح را ثبت و فهرست نتیجه را بررسی کنید. پیش از تغییر سیاست دریافت، نمونه محتوا را اعتبارسنجی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/schemas/page.tsx)

## تیم‌های مدیریتی

`/admin-teams`

تیم بسازید، آن را انتخاب کنید، کاربران را با نقش تیم اضافه کنید و جزئیات یا عضویت را تغییر دهید. پیش از حذف تیم دسترسی‌ها را بررسی کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/admin-teams/page.tsx)

## کاربران مدیر

`/admin-users`

کاربر را با ایمیل، نام، نقش و وضعیت ایجاد یا دعوت کنید. توکن دعوت را محرمانه نگه دارید. پس از بررسی نیازهای دسترسی حساب را ویرایش یا حذف کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/admin-users/page.tsx)

## تنظیمات

`/settings`

ساعت‌های نگهداری و محدودیت نرخ دریافت بر حسب رویداد در ثانیه را بررسی کنید. فقط پس از ارزیابی نیازهای نگهداری و ترافیک ذخیره کنید.

[کد صفحه](https://github.com/arashrasoulzadeh/eye/blob/main/admin/src/app/%28dashboard%29/settings/page.tsx)
