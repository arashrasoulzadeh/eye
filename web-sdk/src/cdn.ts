import { EyeClient, type EyeClientOptions } from "./index";

/**
 * Entry point for the CDN/`<script>`-tag build. Wraps EyeClient with a
 * global `window.Eye` surface so a page can drop in a single <script src>
 * (no bundler/import required) and call `Eye.init({ apiKey, tenant })`.
 */
export type AutoTrackOptions = {
  /** Fires one "page_view" event right after the session is established, with url/path/referrer/title. Default: true. */
  pageView?: boolean;
  /** Fires a "click" event for clicks on <a>/<button>/[data-eye-event] elements, with tag/id/class/text(/href). Default: true. */
  clicks?: boolean;
  /** Fires a "page_exit" event (once) with time-on-page in ms, on tab hide or unload. Default: true. */
  pageExit?: boolean;
  /** Fires a "form_submit" event when any <form> on the page is submitted, with the form's id/name. Default: true. */
  formSubmit?: boolean;
  /** Fires "scroll_depth" once per 25/50/75/100% milestone reached. Default: true. */
  scrollDepth?: boolean;
  /** Fires "rage_click" when the same element is clicked 3+ times within 1s (frustration signal). Default: true. */
  rageClicks?: boolean;
  /** Fires "outbound_click" (in addition to "click") when a link navigates to a different hostname. Default: true. */
  outboundLinks?: boolean;
  /** Fires "js_error" on uncaught errors and unhandled promise rejections. Default: true. */
  jsErrors?: boolean;
  /** Fires "copy" with the (truncated) copied text when the user copies from the page. Default: true. */
  copyText?: boolean;
  /** Fires "video_play"/"video_pause"/"video_ended" for <video> elements on the page. Default: true. */
  video?: boolean;
  /** Fires "file_download" when a link to a downloadable file (pdf/zip/csv/doc/...) is clicked. Default: true. */
  fileDownloads?: boolean;
  /** Fires "field_change" when a text/select input is blurred after changing, with the field's name/id (never its value). Default: true. */
  fieldChanges?: boolean;
  /** Fires "tab_focus"/"tab_blur" when the browser tab gains/loses focus. Default: true. */
  tabFocus?: boolean;
  /** Fires "print" when the user prints the page. Default: true. */
  print?: boolean;
  /** Fires "dead_click" for a click on a non-interactive element that causes no DOM change within 500ms. Default: true. */
  deadClicks?: boolean;
  /** Fires "search" when a [type=search] or [role=search] input's form is submitted, with the query. Default: true. */
  search?: boolean;
  /** Fires "element_viewed" the first time an element tagged data-eye-view scrolls into view. Default: true. */
  elementVisibility?: boolean;
  /** Fires "idle" after `idleTimeoutMs` (default 30000) of no mouse/keyboard/scroll activity. Default: true. */
  idle?: boolean;
  /** Fires "online"/"offline" when the browser's connectivity status changes. Default: true. */
  connectivity?: boolean;
};

const AUTO_TRACK_DEFAULTS: Required<AutoTrackOptions> = {
  pageView: true,
  clicks: true,
  pageExit: true,
  formSubmit: true,
  scrollDepth: true,
  rageClicks: true,
  outboundLinks: true,
  jsErrors: true,
  copyText: true,
  video: true,
  fileDownloads: true,
  fieldChanges: true,
  tabFocus: true,
  print: true,
  deadClicks: true,
  search: true,
  elementVisibility: true,
  idle: true,
  connectivity: true,
};

const DOWNLOAD_EXTENSIONS = /\.(pdf|zip|rar|7z|csv|xlsx?|docx?|pptx?|dmg|exe|pkg|tar|gz|mp3|mp4)$/i;

export type EyeInitOptions = Omit<EyeClientOptions, "tenantId"> & {
  /** Tenant ID. Also accepted as `tenantId` for parity with EyeClientOptions. */
  tenant?: string;
  tenantId?: string;
  apiKey: string;
  endpoint?: string;
  /**
   * Automatic event tracking, on by default. Pass `false` to disable all of
   * it, or an object to toggle individual events (see AutoTrackOptions).
   */
  autoTrack?: boolean | AutoTrackOptions;
  /** Inactivity threshold for the "idle" auto-event, in ms. Default: 30000. */
  idleTimeoutMs?: number;
};

function resolveAutoTrack(opt: boolean | AutoTrackOptions | undefined): Required<AutoTrackOptions> {
  if (opt === false) {
    return Object.fromEntries(Object.keys(AUTO_TRACK_DEFAULTS).map((k) => [k, false])) as Required<AutoTrackOptions>;
  }
  if (opt === true || opt === undefined) return { ...AUTO_TRACK_DEFAULTS };
  return { ...AUTO_TRACK_DEFAULTS, ...opt };
}

/** Best-effort "closest interesting ancestor" click target, without a DOM polyfill dependency. */
function closestTrackable(el: EventTarget | null): HTMLElement | null {
  let node = el as HTMLElement | null;
  while (node && node !== document.body) {
    if (node.hasAttribute?.("data-eye-event") || node.tagName === "A" || node.tagName === "BUTTON") return node;
    node = node.parentElement;
  }
  return null;
}

class EyeGlobal {
  private client: EyeClient | null = null;
  private teardown: Array<() => void> = [];
  private traits: Record<string, any> | undefined;
  private userId: string | undefined;

  /**
   * Initializes the global client. Safe to call once per page load; calling
   * it again replaces the previous client (its auto-flush timer and any
   * auto-track listeners are torn down first).
   */
  init(opts: EyeInitOptions): EyeClient {
    const tenantId = opts.tenant ?? opts.tenantId;
    if (!tenantId) throw new Error("[eye] init() requires `tenant` (or `tenantId`)");
    if (!opts.apiKey) throw new Error("[eye] init() requires `apiKey`");
    this.destroy();

    const endpoint = opts.endpoint ?? EyeGlobal.defaultEndpoint();
    const debug = opts.debug ?? false;
    const client = new EyeClient({ ...opts, tenantId, endpoint });
    this.client = client;

    const auto = resolveAutoTrack(opts.autoTrack);
    client.ensureSession(endpoint!, opts.apiKey)
      .then(() => this.setupAutoTrack(auto, debug, opts.idleTimeoutMs ?? 30000))
      .catch((err) => {
        if (debug) console.error("[eye] ensureSession failed", err);
      });

    return client;
  }

  /**
   * Not a DOM auto-event: records a user identity + traits that get merged
   * into every subsequent track() call's props (as userId/traits.*) until
   * destroy()/re-init(). Call it any time after init(), e.g. on login.
   */
  identify(userId: string, traits?: Record<string, any>) {
    this.userId = userId;
    this.traits = traits;
  }

  track(name: string, props?: Record<string, any>, payload?: Record<string, any>) {
    this.require().track(name, this.withIdentity(props), payload);
  }

  flush() {
    return this.require().flush();
  }

  destroy() {
    for (const off of this.teardown.splice(0)) off();
    this.client?.destroy();
    this.client = null;
    this.userId = undefined;
    this.traits = undefined;
  }

  private withIdentity(props?: Record<string, any>): Record<string, any> | undefined {
    if (!this.userId) return props;
    return { ...props, userId: this.userId, ...this.traits };
  }

  private require(): EyeClient {
    if (!this.client) throw new Error("[eye] call Eye.init({ apiKey, tenant }) first");
    return this.client;
  }

  private setupAutoTrack(auto: Required<AutoTrackOptions>, debug: boolean, idleTimeoutMs: number) {
    if (typeof window === "undefined" || typeof document === "undefined") return;
    const safeTrack = (name: string, props?: Record<string, any>) => {
      try {
        this.client?.track(name, this.withIdentity(props));
      } catch (err) {
        if (debug) console.warn(`[eye] auto-track "${name}" failed`, err);
      }
    };

    if (auto.pageView) {
      safeTrack("page_view", {
        url: window.location.href,
        path: window.location.pathname,
        referrer: document.referrer,
        title: document.title,
      });
    }

    if (auto.clicks || auto.outboundLinks || auto.rageClicks || auto.fileDownloads || auto.deadClicks) {
      const rageTracker = new Map<HTMLElement, { count: number; firstAt: number }>();
      const onClick = (e: MouseEvent) => {
        const el = closestTrackable(e.target);

        if (!el && auto.deadClicks) {
          // Click didn't hit an <a>/<button>/[data-eye-event] — only a real
          // candidate for "dead" if the raw target isn't itself interactive
          // (input/select/textarea/label, or inside a form), to avoid noise.
          const raw = e.target as HTMLElement | null;
          const interactiveTag = /^(INPUT|SELECT|TEXTAREA|LABEL)$/i;
          if (raw && !interactiveTag.test(raw.tagName) && !raw.closest("form")) {
            const before = document.body.innerHTML.length;
            setTimeout(() => {
              if (document.body.innerHTML.length === before) {
                safeTrack("dead_click", { tag: raw.tagName.toLowerCase(), id: raw.id || undefined });
              }
            }, 500);
          }
        }
        if (!el) return;

        const href = el.tagName === "A" ? (el as HTMLAnchorElement).href : undefined;
        const isOutbound = !!href && (() => {
          try {
            return new URL(href, window.location.href).hostname !== window.location.hostname;
          } catch {
            return false;
          }
        })();
        const isDownload = !!href && DOWNLOAD_EXTENSIONS.test(href);

        if (auto.clicks) {
          safeTrack("click", {
            tag: el.tagName.toLowerCase(),
            id: el.id || undefined,
            class: el.className || undefined,
            text: (el.textContent || "").trim().slice(0, 120) || undefined,
            href,
            event: el.getAttribute("data-eye-event") || undefined,
          });
        }
        if (auto.outboundLinks && isOutbound) safeTrack("outbound_click", { href });
        if (auto.fileDownloads && isDownload) safeTrack("file_download", { href });

        if (auto.rageClicks) {
          const now = Date.now();
          const prior = rageTracker.get(el);
          if (prior && now - prior.firstAt < 1000) {
            prior.count++;
            if (prior.count === 3) safeTrack("rage_click", { tag: el.tagName.toLowerCase(), id: el.id || undefined });
          } else {
            rageTracker.set(el, { count: 1, firstAt: now });
          }
        }
      };
      document.addEventListener("click", onClick, true);
      this.teardown.push(() => document.removeEventListener("click", onClick, true));
    }

    if (auto.formSubmit) {
      const onSubmit = (e: SubmitEvent) => {
        const form = e.target as HTMLFormElement;
        safeTrack("form_submit", { formId: form.id || undefined, formName: form.name || undefined });
      };
      document.addEventListener("submit", onSubmit, true);
      this.teardown.push(() => document.removeEventListener("submit", onSubmit, true));
    }

    if (auto.scrollDepth) {
      const milestones = [25, 50, 75, 100];
      const fired = new Set<number>();
      const onScroll = () => {
        const doc = document.documentElement;
        const scrollable = doc.scrollHeight - doc.clientHeight;
        const pct = scrollable > 0 ? Math.round(((window.scrollY || doc.scrollTop) / scrollable) * 100) : 100;
        for (const m of milestones) {
          if (pct >= m && !fired.has(m)) {
            fired.add(m);
            safeTrack("scroll_depth", { percent: m });
          }
        }
      };
      window.addEventListener("scroll", onScroll, { passive: true });
      this.teardown.push(() => window.removeEventListener("scroll", onScroll));
    }

    if (auto.jsErrors) {
      const onError = (e: ErrorEvent) => {
        safeTrack("js_error", { message: e.message, source: e.filename, line: e.lineno, col: e.colno });
      };
      const onRejection = (e: PromiseRejectionEvent) => {
        safeTrack("js_error", { message: String(e.reason), kind: "unhandled_rejection" });
      };
      window.addEventListener("error", onError);
      window.addEventListener("unhandledrejection", onRejection);
      this.teardown.push(() => {
        window.removeEventListener("error", onError);
        window.removeEventListener("unhandledrejection", onRejection);
      });
    }

    if (auto.copyText) {
      const onCopy = () => {
        const text = window.getSelection?.()?.toString().trim();
        if (text) safeTrack("copy", { text: text.slice(0, 200) });
      };
      document.addEventListener("copy", onCopy);
      this.teardown.push(() => document.removeEventListener("copy", onCopy));
    }

    if (auto.video) {
      const bind = (video: HTMLVideoElement) => {
        const onPlay = () => safeTrack("video_play", { src: video.currentSrc, time: video.currentTime });
        const onPause = () => safeTrack("video_pause", { src: video.currentSrc, time: video.currentTime });
        const onEnded = () => safeTrack("video_ended", { src: video.currentSrc });
        video.addEventListener("play", onPlay);
        video.addEventListener("pause", onPause);
        video.addEventListener("ended", onEnded);
        this.teardown.push(() => {
          video.removeEventListener("play", onPlay);
          video.removeEventListener("pause", onPause);
          video.removeEventListener("ended", onEnded);
        });
      };
      document.querySelectorAll("video").forEach((v) => bind(v as HTMLVideoElement));
      const observer = new MutationObserver((mutations) => {
        for (const m of mutations) {
          m.addedNodes.forEach((n) => {
            if (n instanceof HTMLVideoElement) bind(n);
            else if (n instanceof HTMLElement) n.querySelectorAll?.("video").forEach((v) => bind(v as HTMLVideoElement));
          });
        }
      });
      observer.observe(document.body, { childList: true, subtree: true });
      this.teardown.push(() => observer.disconnect());
    }

    if (auto.fieldChanges) {
      const onChange = (e: Event) => {
        const el = e.target as HTMLElement;
        if (!/^(INPUT|SELECT|TEXTAREA)$/.test(el.tagName)) return;
        const type = (el as HTMLInputElement).type;
        if (type === "password") return; // never touch sensitive fields
        safeTrack("field_change", { name: (el as HTMLInputElement).name || undefined, id: el.id || undefined, type });
      };
      document.addEventListener("change", onChange, true);
      this.teardown.push(() => document.removeEventListener("change", onChange, true));
    }

    if (auto.search) {
      const onSubmit = (e: SubmitEvent) => {
        const form = e.target as HTMLFormElement;
        const input = form.querySelector<HTMLInputElement>('input[type="search"], [role="search"] input, input[name*="search" i], input[name*="q" i]');
        if (input) safeTrack("search", { query: input.value.slice(0, 200) });
      };
      document.addEventListener("submit", onSubmit, true);
      this.teardown.push(() => document.removeEventListener("submit", onSubmit, true));
    }

    if (auto.tabFocus) {
      const onFocus = () => safeTrack("tab_focus");
      const onBlur = () => safeTrack("tab_blur");
      window.addEventListener("focus", onFocus);
      window.addEventListener("blur", onBlur);
      this.teardown.push(() => {
        window.removeEventListener("focus", onFocus);
        window.removeEventListener("blur", onBlur);
      });
    }

    if (auto.print) {
      const onPrint = () => safeTrack("print");
      window.addEventListener("beforeprint", onPrint);
      this.teardown.push(() => window.removeEventListener("beforeprint", onPrint));
    }

    if (auto.elementVisibility) {
      const seen = new WeakSet<Element>();
      const observer = new IntersectionObserver((entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && !seen.has(entry.target)) {
            seen.add(entry.target);
            safeTrack("element_viewed", {
              id: entry.target.id || undefined,
              view: entry.target.getAttribute("data-eye-view") || undefined,
            });
          }
        }
      });
      document.querySelectorAll("[data-eye-view]").forEach((el) => observer.observe(el));
      this.teardown.push(() => observer.disconnect());
    }

    if (auto.idle) {
      let timer: ReturnType<typeof setTimeout> | undefined;
      let isIdle = false;
      const onActivity = () => {
        if (isIdle) {
          isIdle = false;
          safeTrack("idle_end");
        }
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => {
          isIdle = true;
          safeTrack("idle", { idleTimeoutMs });
        }, idleTimeoutMs);
      };
      const activityEvents = ["mousemove", "keydown", "scroll", "click"] as const;
      activityEvents.forEach((ev) => window.addEventListener(ev, onActivity, { passive: true }));
      onActivity();
      this.teardown.push(() => {
        activityEvents.forEach((ev) => window.removeEventListener(ev, onActivity));
        if (timer) clearTimeout(timer);
      });
    }

    if (auto.connectivity) {
      const onOnline = () => safeTrack("online");
      const onOffline = () => safeTrack("offline");
      window.addEventListener("online", onOnline);
      window.addEventListener("offline", onOffline);
      this.teardown.push(() => {
        window.removeEventListener("online", onOnline);
        window.removeEventListener("offline", onOffline);
      });
    }

    if (auto.pageExit) {
      const start = Date.now();
      let fired = false;
      const fireExit = () => {
        if (fired) return;
        fired = true;
        safeTrack("page_exit", { durationMs: Date.now() - start });
        this.client?.flush().catch(() => {});
      };
      const onVisibility = () => {
        if (document.visibilityState === "hidden") fireExit();
      };
      document.addEventListener("visibilitychange", onVisibility);
      window.addEventListener("pagehide", fireExit);
      this.teardown.push(() => {
        document.removeEventListener("visibilitychange", onVisibility);
        window.removeEventListener("pagehide", fireExit);
      });
    }
  }

  /** Reads `data-endpoint` off the loading <script> tag, if present. */
  private static defaultEndpoint(): string | undefined {
    if (typeof document === "undefined") return undefined;
    const script = document.currentScript as HTMLScriptElement | null;
    return script?.dataset?.endpoint;
  }
}

const Eye = new EyeGlobal();

declare global {
  interface Window {
    Eye: EyeGlobal;
  }
}

if (typeof window !== "undefined") {
  window.Eye = Eye;
}

export default Eye;
