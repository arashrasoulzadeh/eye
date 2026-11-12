"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Language = "en" | "fa";
type LanguageContextValue = { language: Language; setLanguage: (language: Language) => void; isPersian: boolean };
const LanguageContext = createContext<LanguageContextValue | null>(null);
const STORAGE_KEY = "eye.admin.language";

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguageState] = useState<Language>("en");
  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === "fa" || stored === "en") setLanguageState(stored);
  }, []);
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

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage();
  const next = language === "fa" ? "en" : "fa";
  return <button type="button" aria-label={next === "fa" ? "Switch to Persian" : "Switch to English"} title={next === "fa" ? "فارسی" : "English"} onClick={() => setLanguage(next)} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-fg-muted hover:bg-surface-2 hover:text-fg"><span aria-hidden="true">{next === "fa" ? "🇮🇷" : "🇬🇧"}</span>{next === "fa" ? "فارسی" : "English"}</button>;
}
