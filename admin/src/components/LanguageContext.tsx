"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Language = "en" | "fa";
type LanguageContextValue = { language: Language; setLanguage: (language: Language) => void; isPersian: boolean };
const LanguageContext = createContext<LanguageContextValue | null>(null);
const STORAGE_KEY = "eye.admin.language";
const TRANSLATIONS: Record<string, string> = {
  "Filter logic": "منطق فیلتر",
  "Add filter": "افزودن فیلتر",
  "Remove filter": "حذف فیلتر",
  "contains": "شامل",
  "Compare previous period": "مقایسه دوره قبل",
  "Correlate with": "همبستگی با",
  "Analysis name": "نام تحلیل",
  "Save analysis": "ذخیره تحلیل",
  "Load analysis": "بارگیری تحلیل",
  "Delete analysis": "حذف تحلیل",
  "Pearson correlation": "همبستگی پیرسون",
  "Previous period": "دوره قبل",
  "Matched events": "رویدادهای منطبق",
  "Change": "تغییر",
  "Matching events": "رویدادهای منطبق",
  "Exact event-type counts": "شمارش دقیق انواع رویداد",
  "Highlight outliers": "نمایش داده‌های پرت",
  "Outliers (3 standard deviations)": "داده‌های پرت (۳ انحراف معیار)",
  "Compare tenant": "مقایسه تنانت",
  "Comparison tenant": "تنانت مقایسه",
  "Pivot": "جدول محوری",
  "Heatmap": "نقشه حرارتی",
  "Stacked bar": "نمودار میله‌ای انباشته",
  "Valid values": "مقادیر معتبر",
  "median": "میانه",
  "p90": "صدک ۹۰",
  "p95": "صدک ۹۵",
  "p99": "صدک ۹۹",
  "stddev": "انحراف معیار",
  "Advanced analysis": "تحلیل پیشرفته",
  "Time bucket": "بازه زمانی",
  "Filter dimension": "فیلتر بعد",
  "Filter operator": "عملگر فیلتر",
  "Filter value": "مقدار فیلتر",
  "Reset analysis": "بازنشانی تحلیل",
  "Dimension keys": "کلیدهای ابعاد",
  "Groups": "گروه‌ها",
  "Total value": "مقدار کل",
  "Display options": "تنظیمات نمایش",
  "Chart limit": "محدودیت نمودار",
  "Percentage": "درصد",
  "Sort by": "مرتب‌سازی بر اساس",
  "Ascending": "صعودی",
  "Descending": "نزولی",
  "hour": "ساعت",
  "day": "روز",
  "month": "ماه",
  "equal": "برابر",
  "not_equal": "نابرابر",
  "exists": "موجود",
  "missing": "ناموجود",
  "value": "مقدار",
  "group": "گروه",
  "Dimensions Explorer": "کاوشگر ابعاد",
  "Analyze event dimensions using the global tenant and time range": "تحلیل ابعاد رویدادها با تنانت و بازه زمانی سراسری",
  "Event types": "انواع رویداد",
  "Add event type": "افزودن نوع رویداد",
  "Remove event type": "حذف نوع رویداد",
  "All event types": "همه انواع رویداد",
  "Group by": "گروه‌بندی بر اساس",
  "Split by": "تفکیک بر اساس",
  "Aggregation": "تجمیع",
  "Select dimension": "انتخاب بعد",
  "Metric dimension": "بعد معیار",
  "Explode arrays": "تفکیک آرایه‌ها",
  "Count each unique array item separately": "شمارش جداگانه هر عضو یکتای آرایه",
  "Events scanned": "رویدادهای بررسی‌شده",
  "Partial results: analysis limits reached": "نتایج جزئی: محدودیت تحلیل اعمال شد",
  "Dimension analysis failed": "تحلیل ابعاد ناموفق بود",
  "count": "شمارش",
  "distinct": "یکتا",
  "sum": "مجموع",
  "avg": "میانگین",
  "min": "کمینه",
  "max": "بیشینه",
  "None": "هیچ‌کدام",
  "Suggestions": "پیشنهادها",
  "Insert suggestion": "درج پیشنهاد",
  "Query tip": "نکته پرس‌وجو",
  "Start with SELECT to explore events.": "برای بررسی رویدادها با SELECT شروع کنید.",
  "Only read-only SELECT queries are supported.": "فقط پرس‌وجوهای خواندنی SELECT پشتیبانی می‌شوند.",
  "Add FROM events to choose your data source.": "برای انتخاب منبع داده FROM events را اضافه کنید.",
  "The available table is events.": "جدول موجود events است.",
  "Use ON to specify how joined events match.": "برای تعیین ارتباط رویدادها از ON استفاده کنید.",
  "Use GROUP BY when selecting dimensions with aggregates.": "هنگام انتخاب ابعاد همراه با توابع تجمیعی از GROUP BY استفاده کنید.",
  "Put text values in single quotes, for example 'answer'.": "مقادیر متنی را در نقل‌قول تکی قرار دهید، مانند 'answer'.",
  "Select specific columns to keep results compact.": "برای نتایج فشرده ستون‌های مشخص را انتخاب کنید.",
  "Add LIMIT 100 to preview a smaller result.": "برای پیش‌نمایش کوچک‌تر LIMIT 100 را اضافه کنید.",
  "Use Ctrl/⌘ + Enter to run your query.": "برای اجرا از Ctrl/⌘ + Enter استفاده کنید.",
  "Run selected tenant query (Ctrl/⌘ + Enter)": "اجرای پرس‌وجوی تنانت انتخاب‌شده (Ctrl/⌘ + Enter)",
  "Restrict events to the dashboard time range": "محدود کردن رویدادها به بازه زمانی داشبورد",
  "Referenced tables": "جدول‌های ارجاع‌شده",
  "Inspect table schema": "مشاهده طرحواره جدول",
  "Only events is available": "فقط جدول رویدادها در دسترس است",
  "Insert field into query": "درج فیلد در پرس‌وجو",
  "Show less": "نمایش کمتر",
  "Show all fields": "نمایش همه فیلدها",
  "Smart query examples": "نمونه‌های هوشمند پرس‌وجو",
  "Active users": "کاربران فعال",
  "Busiest sessions": "پرترافیک‌ترین نشست‌ها",
  "Latest events": "آخرین رویدادها",
  "Sort by column": "مرتب‌سازی بر اساس ستون",
  "Download all returned rows as CSV": "دریافت همه ردیف‌های نتیجه به صورت CSV",
  "Download all returned rows as JSON": "دریافت همه ردیف‌های نتیجه به صورت JSON",
  "Query Explorer": "کاوشگر پرس‌وجو",
  "Select a tenant before running a query.": "پیش از اجرای پرس‌وجو یک تنانت انتخاب کنید.",
  "Read-only event SQL": "SQL فقط‌خواندنی رویدادها",
  "Query failed": "پرس‌وجو ناموفق بود",
  "Query history": "تاریخچه پرس‌وجو",
  "Saved queries": "پرس‌وجوهای ذخیره‌شده",
  "Save query": "ذخیره پرس‌وجو",
  "Query name": "نام پرس‌وجو",
  "Export CSV": "خروجی CSV",
  "Export JSON": "خروجی JSON",
  "Results": "نتایج",
  "Rows": "ردیف‌ها",
  "Execution time (ms)": "زمان اجرا (میلی‌ثانیه)",
  "Results limited to 1000 rows.": "نتایج به ۱۰۰۰ ردیف محدود شده‌اند.",
  "Running query…": "در حال اجرای پرس‌وجو…",
  "Schema explorer": "کاوشگر طرحواره",
  "Custom properties": "ویژگی‌های سفارشی",
  "Use the global time range": "استفاده از بازه زمانی سراسری",
  "No results": "بدون نتیجه",
  "Query examples": "نمونه‌های پرس‌وجو",
  "Event counts": "تعداد رویدادها",
  "Answer streak": "زنجیره پاسخ‌ها",
  "Storage unavailable": "ذخیره‌سازی در دسترس نیست",
  "Pinned funnel": "قیف سنجاق‌شده",
  "Funnel name": "نام قیف",
  "Configure funnel": "تنظیم قیف",
  "Open pinned funnel": "باز کردن قیف سنجاق‌شده",
  "No funnel pinned for this tenant.": "برای این تنانت قیفی سنجاق نشده است.",
  "Loading pinned funnel…": "در حال بارگذاری قیف سنجاق‌شده…",
  "Could not load pinned funnel.": "بارگذاری قیف سنجاق‌شده ناموفق بود.",
  "Could not save pinned funnel.": "ذخیره قیف سنجاق‌شده ناموفق بود.",
  "Could not remove pinned funnel.": "برداشتن قیف سنجاق‌شده ناموفق بود.",
  "Pin to dashboard": "سنجاق به داشبورد",
  "Update pinned funnel": "به‌روزرسانی قیف سنجاق‌شده",
  "Load pinned funnel": "بارگذاری قیف سنجاق‌شده",
  "Unpin funnel": "برداشتن سنجاق قیف",
  "Funnel pinned to dashboard.": "قیف به داشبورد سنجاق شد.",
  "Funnel removed from dashboard.": "قیف از داشبورد برداشته شد.",
  "Change vs previous period": "تغییر نسبت به دوره قبل",
  "percentage points": "واحد درصد",
  "Activity overview": "خلاصه فعالیت",
  "Technical metrics": "شاخص‌های فنی",
  "Failed to load dashboard metrics": "بارگذاری شاخص‌های داشبورد ناموفق بود.",
  "Estimated from distinct sessions": "برآورد بر اساس نشست‌های یکتا",
  "All tenants · Last 5 minutes": "همه تنانت‌ها · ۵ دقیقه اخیر",
  "All tenants · 7-day return rate": "همه تنانت‌ها · نرخ بازگشت ۷ روزه",
  "Hourly buckets": "بازه‌های ساعتی",
  "Cache age": "عمر کش",
  "Last minute": "۱ دقیقه اخیر",
  "Last 4 minutes": "۴ دقیقه اخیر",
  "Last hour": "۱ ساعت اخیر",
  "Last 24 hours": "۲۴ ساعت اخیر",
  "Last 7 days": "۷ روز اخیر",
  "Last 30 days": "۳۰ روز اخیر",
  "Last {count} minutes": "{count} دقیقه اخیر",
  "Last {count} hours": "{count} ساعت اخیر",
  "Last {count} days": "{count} روز اخیر",
  "Last {count} weeks": "{count} هفته اخیر",
  "Last {count} months": "{count} ماه اخیر",
  "Share of first-stage sessions": "سهم از نشست‌های مرحله اول",
  "Funnel stages": "مراحل قیف",
  "Largest drop-off": "بیشترین ریزش",
  "Overall conversion": "نرخ تبدیل کل",
  "Final stage ÷ first stage": "مرحله آخر تقسیم بر مرحله اول",
  "Add ordered events or use a smart suggestion to analyze a session journey.": "برای تحلیل مسیر نشست، رویدادها را به ترتیب اضافه کنید یا از پیشنهاد هوشمند استفاده کنید.",
  "Linked time range": "بازه زمانی پیوند",
  "Use smart suggestion": "استفاده از پیشنهاد هوشمند",
  "No funnel suggestion for this range.": "برای این بازه پیشنهادی برای قیف پیدا نشد.",
  "Move stage earlier": "انتقال مرحله به قبل",
  "Move stage later": "انتقال مرحله به بعد",
  "Counts include raw and generated event records.": "شمارش شامل رکوردهای خام و رویدادهای تولیدشده است.",
  "Top events by volume": "پرتکرارترین رویدادها",
  "Loading event counts…": "در حال بارگذاری تعداد رویدادها…",
  "Could not load event counts.": "بارگذاری تعداد رویدادها ناموفق بود.",
  "Sessions entering": "نشست‌های ورودی",
  "Sessions completing": "نشست‌های تکمیل‌کننده",
  "Sessions reaching stage 1": "نشست‌هایی که به مرحله اول رسیده‌اند",
  "Sessions reaching the final stage": "نشست‌هایی که به مرحله آخر رسیده‌اند",
  "Dropped sessions": "نشست‌های ریزش‌کرده",
  "Conversion to next stage": "نرخ عبور به مرحله بعد",
  "Download a training batch to share privately. Inference runs locally with the released model.": "یک فایل آموزشی برای اشتراک‌گذاری خصوصی دانلود کنید. مدل منتشرشده روی همین سرور اجرا می‌شود.",
  Created: "زمان ساخت",
  "first time visitor": "بازدیدکننده بار اول", "frequent visitor": "بازدیدکننده پرتکرار",
  returning: "بازگشتی", buyer: "خریدار", "repeat buyer": "خریدار تکراری",
  "new player": "بازیکن جدید", "returning player": "بازیکن بازگشتی",
  "rewarded ad regular": "تماشاگر مکرر تبلیغ جایزه‌دار",
  converted: "تبدیل‌شده", checkout: "تسویه‌حساب", "error encountered": "مواجهه با خطا",
  "entry point": "نقطه ورود", navigation: "پیمایش", engagement: "تعامل",
  "cart add": "افزودن به سبد", "purchase confirmed": "خرید تأییدشده",
  "question answered": "سؤال پاسخ داده شد", "question missed": "سؤال از دست رفت",
  "precise hit": "ضربه دقیق", "combo reached": "کمبو ثبت شد", "life lost": "جان از دست رفت",
  "Tenant usage is temporarily unavailable.": "اطلاعات مصرف تنانت موقتاً در دسترس نیست.",
  browsing: "مرور", gameplay: "بازی",
  "Tenant is outside the selected scope.": "تنانت خارج از محدوده انتخاب‌شده است.",
  "Intelligence tags": "برچسب‌های هوشمند",
  "Global model": "مدل سراسری",
  "Download latest model": "دانلود آخرین مدل",
  "Installed model": "مدل نصب‌شده",
  "Latest release": "آخرین نسخه منتشرشده",
  "Up to date": "به‌روز",
  "New model available": "مدل جدید در دسترس است",
  "No model release available.": "هنوز مدلی منتشر نشده است.",
  "Could not check model releases.": "بررسی نسخه‌های مدل ناموفق بود.",
  "Could not download the model.": "دانلود مدل ناموفق بود.",
  "Download training CSV": "دانلود فایل CSV آموزشی",
  "Last successful model": "آخرین مدل اجراشده با موفقیت",
  "No successful inference yet.": "هنوز اجرای موفقی ثبت نشده است.",
  "Inference log": "گزارش اجرای مدل",
  "Loading inference log…": "در حال بارگذاری گزارش اجرای مدل…",
  "Could not load inference log.": "بارگذاری گزارش اجرای مدل ناموفق بود.",
  "No inference runs yet.": "هنوز اجرای مدل ثبت نشده است.",
  "Model version": "نسخه مدل",
  "Inference failed": "اجرای مدل ناموفق بود",
  "Inference completed": "اجرای مدل پایان یافت",
  Dashboard: "داشبورد", "Live Stream": "پخش زنده", Analytics: "تحلیل‌ها", Graphs: "نمودارها", Questions: "سؤال‌ها",
  Funnels: "قیف‌ها", Retention: "بازگشت", "Health Center": "مرکز سلامت", Tenants: "تنانت‌ها", Sessions: "نشست‌ها",
  Users: "کاربران", Events: "رویدادها", Alerts: "هشدارها", Operations: "عملیات", "Admin Users": "کاربران مدیر",
  Intelligence: "هوش کاربران", "Model Training": "آموزش مدل", "Model Registry": "مخزن مدل",
  "Admin Teams": "تیم‌های مدیر", "Test Event": "رویداد آزمایشی", "CDN Test": "آزمایش CDN", Schemas: "طرحواره‌ها",
  Cohorts: "گروه‌ها", "Feature Flags": "پرچم‌های قابلیت", Aggregations: "تجمیع‌ها", Anomalies: "ناهنجاری‌ها",
  "Audit Log": "گزارش ممیزی", "Dead Letters": "صف خطا", Replay: "بازپخش", Experiments: "آزمایش‌ها",
  Settings: "تنظیمات", "SLO & Error Budget": "SLO و بودجه خطا", More: "بیشتر", "Tenant scope": "محدوده تنانت",
  "Global time": "زمان سراسری", Release: "نسخه", "Log out": "خروج", Protected: "محافظت‌شده", "Admin console": "کنسول مدیریت",
  Admin: "مدیریت", "Funnel Analysis": "تحلیل قیف", "Retention Analysis": "تحلیل بازگشت", "Event Replay": "بازپخش رویداد",
  Filters: "فیلترها", Apply: "اعمال", Refresh: "تازه‌سازی", Close: "بستن", Cancel: "لغو", Save: "ذخیره", Create: "ایجاد", Delete: "حذف", Edit: "ویرایش", Details: "جزئیات", Copy: "کپی", Inspect: "بررسی", Actions: "عملیات",
  "Select tenant": "انتخاب تنانت", "All tenants": "همه تنانت‌ها", Tenant: "تنانت", "Tenant ID": "شناسه تنانت", "Session ID": "شناسه نشست", "Event name": "نام رویداد", "Global time range": "بازه زمانی سراسری", "Global range": "بازه سراسری",
  "How to use this page": "راهنمای استفاده از این صفحه", "What am I seeing?": "چه چیزی می‌بینم؟", "Analytics help": "راهنمای تحلیل‌ها", "Users data help": "راهنمای داده کاربران", "Data:": "داده:", "Actions:": "عملیات:", "Safety:": "ایمنی:",
  "Online users": "کاربران آنلاین", "Active sessions": "نشست‌های فعال", "Events / hour": "رویداد در ساعت", "Events / minute": "رویداد در دقیقه", "Events / second": "رویداد در ثانیه",
  "Run query": "اجرای پرس‌وجو", "Add panel": "افزودن پنل", "Copy share link": "کپی لینک اشتراک‌گذاری", "Line chart": "نمودار خطی", "Area chart": "نمودار ناحیه‌ای", "Bar chart": "نمودار میله‌ای", "Pie chart": "نمودار دایره‌ای", Table: "جدول",
  "Run funnel": "اجرای قیف", "Refresh smart suggestion": "تازه‌سازی پیشنهاد هوشمند", "Smart funnel diagram": "نمودار هوشمند قیف", "Replay events": "بازپخش رویدادها", "Replay range": "بازه بازپخش", Payload: "بار داده", Dimensions: "ابعاد", Source: "منبع", Time: "زمان", View: "نما", Cards: "کارت‌ها", JSON: "JSON", Expand: "باز کردن", Compact: "فشرده", Pause: "مکث", Resume: "ادامه",
  "Add member": "افزودن عضو", "Assign variant": "اختصاص نسخه", Active: "فعال", Absence: "غیبت", Analyst: "تحلیل‌گر", Owner: "مالک", Invited: "دعوت‌شده", Members: "اعضا", Role: "نقش", Status: "وضعیت", Name: "نام", Email: "ایمیل", Editor: "ویرایشگر",
  "Admin user help": "راهنمای کاربر مدیر", "Alert help": "راهنمای هشدارها", "Audit log help": "راهنمای گزارش ممیزی", "Cohort help": "راهنمای گروه‌ها", "Schema help": "راهنمای طرحواره", "Settings help": "راهنمای تنظیمات", "Session data help": "راهنمای داده نشست", "Session detail help": "راهنمای جزئیات نشست",
  "Add a tenant": "افزودن تنانت", "Create a tenant first": "ابتدا یک تنانت ایجاد کنید", "Create tenant": "ایجاد تنانت", "Create rule": "ایجاد قانون", "Save rule": "ذخیره قانون", "Save quotas": "ذخیره سهمیه‌ها", "Delete data": "حذف داده", "Delete tenant analytics data": "حذف داده‌های تحلیلی تنانت", "Backfill": "تکمیل گذشته", "Materialize": "مادی‌سازی",
  "Back to sessions": "بازگشت به نشست‌ها", "Back to tenants": "بازگشت به تنانت‌ها", Previous: "قبلی", Next: "بعدی", Remove: "حذف", Rename: "تغییر نام", Send: "ارسال", Enable: "فعال‌سازی", "Enable flag": "فعال‌سازی پرچم", "Download CSV": "دانلود CSV", "Print report": "چاپ گزارش",
  "Event detail": "جزئیات رویداد", "Event explorer": "کاوشگر رویداد", "Event ID: ": "شناسه رویداد: ", "Full event": "رویداد کامل", "Client time": "زمان کلاینت", "Last seen": "آخرین مشاهده", Location: "مکان", "Browser / IP": "مرورگر / IP", IP: "IP", Session: "نشست", "Session based": "مبتنی بر نشست",
  "Loading tenants…": "در حال بارگذاری تنانت‌ها…", "Loading tenant…": "در حال بارگذاری تنانت…", "Loading event…": "در حال بارگذاری رویداد…", "Loading session…": "در حال بارگذاری نشست…", "Loading live events…": "در حال بارگذاری رویدادهای زنده…", "Loading questions…": "در حال بارگذاری سؤال‌ها…", "Loading panel…": "در حال بارگذاری پنل…", "Loading retention data…": "در حال بارگذاری داده بازگشت…",
  "No data yet.": "هنوز داده‌ای وجود ندارد.", "No events yet.": "هنوز رویدادی وجود ندارد.", "No events sent yet.": "هنوز رویدادی ارسال نشده است.", "No events match the current filters.": "رویدادی با فیلترهای فعلی مطابقت ندارد.", "No questions exist for this tenant. Create one on the Questions page first.": "برای این تنانت سؤالی وجود ندارد. ابتدا در صفحه سؤال‌ها یکی ایجاد کنید.", "No sessions — create one": "نشستی وجود ندارد — یکی ایجاد کنید", "No tenants yet": "هنوز تنانتی وجود ندارد", "No users found for this tenant.": "کاربری برای این تنانت یافت نشد.",
  "Newest first": "جدیدترین ابتدا", "Oldest first": "قدیمی‌ترین ابتدا", "Record details": "جزئیات رکورد", "Row details": "جزئیات ردیف", "No matching page.": "صفحه‌ای مطابق یافت نشد", "Not implemented yet": "هنوز پیاده‌سازی نشده است", "No members in this team.": "این تیم عضوی ندارد.",
  "Availability objective": "هدف دسترس‌پذیری", Availability: "دسترس‌پذیری", "Error budget": "بودجه خطا", "Action required": "نیازمند اقدام", "Within SLO": "در محدوده SLO", "Current anomalies": "ناهنجاری‌های فعلی", "Dead letters": "صف خطا", "Live event count": "تعداد رویداد زنده", "Live session": "نشست زنده",
  "Review system health": "بررسی سلامت سیستم", "Investigate trends": "بررسی روندها", "Watch live traffic": "مشاهده ترافیک زنده", "Recommended next steps": "گام‌های پیشنهادی بعدی", "Recommended workflow": "روند کاری پیشنهادی", "Clear delivery debt": "پاک‌سازی بدهی ارسال", "Review anomalies": "بررسی ناهنجاری‌ها",
  "Retention (hours)": "بازگشت (ساعت)", "Min count": "حداقل تعداد", Count: "تعداد", Conversion: "تبدیل", "Conversion rate": "نرخ تبدیل", "Completed sequence": "دنباله تکمیل‌شده", "Group workspace users and assign team-scoped roles.": "کاربران فضای کاری را گروه‌بندی و نقش‌های تیمی اختصاص دهید.",
  "Register and version event payload schemas per tenant.": "طرحواره‌های بار داده رویداد را برای هر تنانت ثبت و نسخه‌بندی کنید.", "Review administrative actions newest first. Filtering, sorting, and pagination are API-backed.": "اقدامات مدیریتی را از جدیدترین بررسی کنید. فیلتر، مرتب‌سازی و صفحه‌بندی از API انجام می‌شود.", "Review statistically unusual event volume detected from recent tenant history.": "حجم غیرعادی رویداد شناسایی‌شده از تاریخچه اخیر تنانت را بررسی کنید.",
};
function initialLanguage(): Language {
  if (typeof window === "undefined") return "en";
  const stored = window.localStorage.getItem(STORAGE_KEY);
  return stored === "fa" || stored === "en" ? stored : "en";
}

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguageState] = useState<Language>(initialLanguage);
  useEffect(() => {
    document.documentElement.lang = language;
    document.documentElement.dir = language === "fa" ? "rtl" : "ltr";
    translateDocument(language);
    if (language === "fa") {
      const observer = new MutationObserver(() => translateDocument(language));
      observer.observe(document.body, { childList: true, characterData: true, subtree: true });
      return () => observer.disconnect();
    }
  }, [language]);
  const value = useMemo(() => ({ language, isPersian: language === "fa", setLanguage: (next: Language) => { setLanguageState(next); window.localStorage.setItem(STORAGE_KEY, next); } }), [language]);
  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}

export function useLanguage() {
  return useContext(LanguageContext) || { language: "en" as Language, isPersian: false, setLanguage: () => undefined };
}

export function translate(value: string, language: Language): string {
  return language === "fa" ? TRANSLATIONS[value] || value : value;
}

export function translateText(value: string, language: Language): string {
  if (language !== "fa") return value;
  return Object.entries(TRANSLATIONS).sort(([a], [b]) => b.length - a.length).reduce((current, [english, persian]) => current.split(english).join(persian), value);
}

const TEXT_TRANSLATIONS = new WeakMap<Text, { original: string; rendered: string }>();
function translateDocument(language: Language) {
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  const nodes: Text[] = [];
  while (walker.nextNode()) nodes.push(walker.currentNode as Text);
  nodes.forEach((node) => {
    const current = node.data;
    if (!current.trim() || node.parentElement?.closest("script,style,code,pre")) return;
    const previous = TEXT_TRANSLATIONS.get(node);
    const original = previous?.rendered === current ? previous.original : current;
    const rendered = translateText(original, language);
    TEXT_TRANSLATIONS.set(node, { original, rendered });
    if (current !== rendered) node.data = rendered;
  });
}

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage();
  const next = language === "fa" ? "en" : "fa";
  return <button type="button" aria-label={next === "fa" ? "Switch to Persian" : "Switch to English"} title={next === "fa" ? "فارسی" : "English"} onClick={() => setLanguage(next)} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-fg-muted hover:bg-surface-2 hover:text-fg"><span aria-hidden="true">{next === "fa" ? "🇮🇷" : "🇬🇧"}</span>{next === "fa" ? "فارسی" : "English"}</button>;
}
