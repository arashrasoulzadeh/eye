# API تحلیل ابعاد

`POST /query-explorer?mode=dimensions`

احراز هویت نشست مدیر لازم است. بدنه JSON حداکثر ۳۲۷۶۸ بایت؛ فیلد ناشناخته رد می‌شود.

| Field | Type | شرح |
|---|---|---|
| `tenant_id` | string | مستأجر الزامی |
| `from_ts, to_ts` | integer | میلی‌ثانیه یونیکس با احتساب مرزها |
| `types` | string[] | نام رویدادها، حداکثر ۱۰۰ |
| `group, split` | string | کلید بعد یا @event |
| `metric` | string | بعد مورد اندازه‌گیری |
| `operation` | string | عملیات تجمیع الزامی |
| `filters` | object[] | کلید، مقدار، حالت؛ حداکثر ۲۰ |
| `filter_logic` | string | and پیش‌فرض یا or |
| `filter_key, filter_value, filter_mode` | string | فیلتر منفرد قدیمی |
| `time_bucket` | string | خالی، ساعت، روز، ماه؛ UTC |
| `explode` | boolean | حذف تکرار اعضای آرایه در هر رویداد |
| `full_range` | boolean | پویش کامل یا مسیر سریع پایگاه داده |
| `summary` | boolean | شمارش پایگاه داده پیش از فیلتر ابعاد |
| `correlate` | string | بعد عددی دوم |
| `discover_correlations` | boolean | کشف همبستگی عددی |
| `drill_group, drill_split` | string/null | انتخاب گروه و تفکیک اختیاری |

## عملیات و فیلترها

`count`, `sum`, `avg`, `min`, `max`, `distinct`, `missing`, `median`, `p90`, `p95`, `p99`, `stddev`.

حالت‌های فیلتر چندگانه: `equal`, `not_equal`, `contains`, `exists`, `missing`.

مسیر تو در تو از JSON Pointer استفاده می‌کند. contains به بزرگی حروف حساس نیست. انحراف معیار از نوع جامعه است. صدک‌ها با درون‌یابی خطی محاسبه می‌شوند.

```json
{"tenant_id":"example","from_ts":1700000000000,"to_ts":1700086400000,"types":["purchase"],"group":"country","metric":"amount","operation":"sum","filters":[],"explode":false}
```

## خروجی و محدودیت‌ها

ردیف‌ها شامل group، split، value، count و valid هستند. خروجی شامل keys، types، profiles، events، correlation، correlation_n، associations، association_scanned، matched، users، sessions، identified_events، scanned، truncated و execution است. شمارش کاربران و نشست‌ها در حالت summary محاسبه می‌شود.

| محدودیت | Value |
|---|---:|
| Timeout | 8 s |
| Default scan | 20000 |
| Returned rows | 1000 |
| Internal groups | 5000 |
| Distinct entries / percentile samples (each) | 200000 |
| Drill events | 100 |
| Association scan | 10000 |
| Association dimensions | 16 |
| Minimum association pairs | 30 |
| Returned associations | 10 |

پرچم truncated را بررسی کنید. همبستگی دلیل علیت نیست. مسیر کامل نیز محدودیت زمان و حافظه دارد.
