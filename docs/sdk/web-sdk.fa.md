# SDK وب

کلاینت TypeScript برای ثبت رویداد در مرورگر، ذخیره نشست و ارسال دسته‌ای داده‌ها.

```bash
cd web-sdk
npm install
npm run build:cdn
```

```typescript
import { EyeClient } from 'eye-sdk';
const eye = new EyeClient({
  tenantId: 'my-tenant',
  endpoint: 'https://api.example.com',
  apiKey: 'tenant-api-key'
});
await eye.ensureSession('https://api.example.com', 'tenant-api-key');
eye.track('page_view', { page: '/pricing' });
eye.identify('user-123', { plan: 'pro' });
await eye.flush();
eye.destroy();
```

## متدها

| متد | کاربرد |
|---|---|
| `ensureSession` | ایجاد یا بازیابی نشست |
| `getSessionId` | شناسه نشست |
| `track` | ثبت رویداد در صف |
| `identify` | افزودن هویت به رویدادهای بعدی |
| `flush` | ارسال داده‌های صف |
| `destroy` | توقف زمان‌بند |
| `BufferLen` | تعداد رویدادهای صف |
| `PeekBuffer` | مشاهده نسخه‌ای از صف |

ظرفیت پیش‌فرض صف ۱۰۰۰، اندازه دسته ۲۰ و فاصله ارسال ۵ ثانیه است. در نسخه CDN ثبت خودکار بازدید، کلیک، فرم، پیمایش و خطا قابل تنظیم است. داده حساس را در ابعاد یا محتوای رویداد ثبت نکنید.


## ثبت خودکار در CDN

تمام گزینه‌های زیر به طور پیش‌فرض فعال هستند. برای غیرفعال کردن همه، مقدار autoTrack را false قرار دهید. متن کپی‌شده و عبارت جستجو ممکن است حساس باشند؛ پیش از فعال‌سازی سیاست حریم خصوصی را بررسی کنید.

| Option | کاربرد |
|---|---|
| `pageView` | بازدید صفحه |
| `clicks` | کلیک |
| `pageExit` | خروج از صفحه |
| `formSubmit` | ارسال فرم |
| `scrollDepth` | عمق پیمایش |
| `rageClicks` | کلیک‌های سریع تکراری |
| `outboundLinks` | پیوند خارجی |
| `jsErrors` | خطای جاوااسکریپت |
| `copyText` | متن کپی‌شده |
| `video` | فعالیت ویدیو |
| `fileDownloads` | دانلود فایل |
| `fieldChanges` | شناسه فیلد بدون مقدار |
| `tabFocus` | تمرکز زبانه |
| `print` | چاپ |
| `deadClicks` | کلیک بدون تعامل |
| `search` | عبارت جستجو |
| `elementVisibility` | مشاهده عنصر برچسب‌دار |
| `idle` | عدم فعالیت |
| `connectivity` | اتصال شبکه |

```javascript
Eye.init({ tenant: 'my-tenant', apiKey: 'tenant-key', endpoint: 'https://api.example.com', autoTrack: { copyText: false, search: false }, idleTimeoutMs: 30000, forceNewSession: false });
```

متدهای رابط سراسری: `init`, `newSession`, `identify`, `track`, `flush`, `getSessionId`, `destroy`.
