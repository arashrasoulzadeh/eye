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
    const translated = translate(trimmed, language);
    node.textContent = language === "fa" ? raw.replace(trimmed, translated) : raw;
  });
}

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage();
  const next = language === "fa" ? "en" : "fa";
  return <button type="button" aria-label={next === "fa" ? "Switch to Persian" : "Switch to English"} title={next === "fa" ? "فارسی" : "English"} onClick={() => setLanguage(next)} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-fg-muted hover:bg-surface-2 hover:text-fg"><span aria-hidden="true">{next === "fa" ? "🇮🇷" : "🇬🇧"}</span>{next === "fa" ? "فارسی" : "English"}</button>;
}
