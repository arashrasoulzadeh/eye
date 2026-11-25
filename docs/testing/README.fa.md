# آزمون و سنجش عملکرد

## آزمون سرور

```bash
go test ./tests/... -v
go test ./tests/server -coverpkg=./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## آزمون SDK و پنل

```bash
(cd web-sdk && npm test)
(cd sdk-go && go test -v)
(cd admin && npm test)
```

آزمون‌ها دریافت رویداد، محدودیت نرخ، نشست، جریان زنده، هشدار، گروه کاربران و رفتار صف SDK را بررسی می‌کنند.

## بنچمارک

```bash
go test ./tests/server -bench=. -benchmem -benchtime=10s
```

نتایج به سخت‌افزار، پایگاه داده و بار وابسته‌اند؛ اعداد نمونه تضمین عملکرد نیستند.
