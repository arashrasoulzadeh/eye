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
await eye.ensureSession();
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
