"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Language = "en" | "fa";
type LanguageContextValue = { language: Language; setLanguage: (language: Language) => void; isPersian: boolean };
const LanguageContext = createContext<LanguageContextValue | null>(null);
const STORAGE_KEY = "eye.admin.language";
const TRANSLATIONS: Record<string, string> = {
  Dashboard: "داشبورد", "Live Stream": "پخش زنده", Analytics: "تحلیل‌ها", Graphs: "نمودارها", Questions: "سؤال‌ها",
  Funnels: "قیف‌ها", Retention: "بازگشت", "Health Center": "مرکز سلامت", Tenants: "تنانت‌ها", Sessions: "نشست‌ها",
  Users: "کاربران", Events: "رویدادها", Alerts: "هشدارها", Operations: "عملیات", "Admin Users": "کاربران مدیر",
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
      observer.observe(document.body, { childList: true, subtree: true });
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

const ORIGINAL_TEXT = new WeakMap<Text, string>();
function translateDocument(language: Language) {
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  const nodes: Text[] = [];
  while (walker.nextNode()) nodes.push(walker.currentNode as Text);
  nodes.forEach((node) => {
    const raw = ORIGINAL_TEXT.get(node) || node.textContent || "";
    ORIGINAL_TEXT.set(node, raw);
    const trimmed = raw.trim();
    if (!trimmed || node.parentElement?.closest("script,style,code,pre")) return;
    if (language === "fa") {
      node.textContent = translateText(raw, language);
    } else node.textContent = raw;
  });
}

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage();
  const next = language === "fa" ? "en" : "fa";
  return <button type="button" aria-label={next === "fa" ? "Switch to Persian" : "Switch to English"} title={next === "fa" ? "فارسی" : "English"} onClick={() => setLanguage(next)} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-fg-muted hover:bg-surface-2 hover:text-fg"><span aria-hidden="true">{next === "fa" ? "🇮🇷" : "🇬🇧"}</span>{next === "fa" ? "فارسی" : "English"}</button>;
}
