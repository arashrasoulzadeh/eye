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

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage();
  const next = language === "fa" ? "en" : "fa";
  return <button type="button" aria-label={next === "fa" ? "Switch to Persian" : "Switch to English"} title={next === "fa" ? "فارسی" : "English"} onClick={() => setLanguage(next)} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-fg-muted hover:bg-surface-2 hover:text-fg"><span aria-hidden="true">{next === "fa" ? "🇮🇷" : "🇬🇧"}</span>{next === "fa" ? "فارسی" : "English"}</button>;
}
