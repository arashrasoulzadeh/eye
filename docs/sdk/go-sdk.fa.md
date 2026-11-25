# SDK گو

کلاینت سمت سرور برای ایجاد نشست، ثبت رویداد و ارسال دسته‌ای.

```bash
go get github.com/arashrasoulzadeh/eye/sdk-go
```

```go
client := eyesdk.NewClient(eyesdk.Options{
    Endpoint: "http://localhost:8080",
    APIKey: "tenant-api-key",
    TenantID: "my-tenant",
})
defer client.Close()
ctx := context.Background()
if _, err := client.EnsureSession(ctx); err != nil {
    return err
}
if err := client.Track("purchase", map[string]any{"amount": 99.99}, map[string]any{"plan": "pro"}); err != nil {
    return err
}
return client.Flush(ctx)
```

## متدها و تنظیمات

`SetSessionID` برای بازیابی نشست، `Identify` برای هویت، `Track` برای صف، `Flush` برای ارسال، `BufferLen` و `PeekBuffer` برای عیب‌یابی هستند. `Close` زمان‌بند را متوقف می‌کند و جایگزین ارسال نهایی نیست.

ظرفیت صف، اندازه دسته، فاصله ارسال، کلاینت HTTP و ثبت گزارش قابل تنظیم هستند. خطاهای ارسال را بررسی کنید.

```bash
cd sdk-go
go test -v
```
