# قرارداد API هوشمندی

مسیر `/intelligence/users` با احراز هویت مدیر و پارامتر action. به جز چهار عمل نخست، tenant_id الزامی است. روش‌های جدول روش پیشنهادی استفاده هستند؛ بعضی شاخه‌های هندلر روش را محدود نمی‌کنند.

| Action | HTTP | قرارداد |
|---|---|---|
| `model-status` | GET | نسخه نصب‌شده و آخرین انتشار |
| `download-model` | POST | نصب انتشار |
| `training-csv` | GET | CSV؛ مستأجر و زمان اختیاری |
| `inference-log` | GET | ۵۰ اجرای اخیر؛ مستأجر و زمان اختیاری |
| `label` | POST | بدنه JSON شامل fingerprint، label، source؛ پاسخ ۲۰۴ |
| `suggestions` | GET | حداکثر ۵۰ پیشنهاد |
| `review` | POST | بدنه JSON شامل fingerprint، label، decision |
| `pipeline` | POST | شناسه retry_id اختیاری؛ اجرای خط لوله |
| `pipeline-status` | GET | ۲۰ اجرای اخیر |
| `readiness` | GET | برچسب پیش‌فرض buyer |
| `training` | GET | برچسب؛ ۲۰ اجرای اخیر |
| `models` | GET | برچسب؛ حداکثر ۱۰۰ مدل |
| `model-layers` | GET | برچسب؛ حداکثر ۱۰۰ لایه |
| `evaluations` | GET | برچسب؛ حداکثر ۱۰۰ ارزیابی |
| `train` | POST | برچسب پیش‌فرض auto؛ خطای ۴۲۲ |
| `model` | GET | وظیفه پیش‌فرض buyer؛ محدوده و مانیفست |
| `evaluation` | GET / POST | خواندن یا ارزیابی آخرین مدل؛ پیش‌فرض behavior |
| `quality` | GET / POST | خواندن یا سنجش کیفیت داده |
| `alerts` | GET | وضعیت؛ حداکثر ۱۰۰ هشدار |
| `resolve-alert` | POST | شناسه الزامی؛ پاسخ ۲۰۴ |
| `promote` | POST | ارتقای مدل؛ پیش‌فرض behavior |
| `rollback` | POST | بازگشت مدل؛ پیش‌فرض behavior |
| `export` | POST | برچسب buyer و محدوده global به‌صورت پیش‌فرض |
| `refresh` | POST | تعداد تازه‌سازی‌شده |

بدون action، فهرست کاربران با limit بازگردانده می‌شود. عملیات تغییر مدل را فقط با مجوز مناسب اجرا کنید.
