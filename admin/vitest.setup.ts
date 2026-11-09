import "@testing-library/jest-dom/vitest";

// jsdom has no ResizeObserver; recharts' ResponsiveContainer needs one to render.
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

// jsdom has no EventSource; LiveCounter/LiveSession components use it for SSE.
if (typeof globalThis.EventSource === "undefined") {
  globalThis.EventSource = class EventSource {
    constructor() {
      this.readyState = 0;
      this.url = "";
      this.onopen = null;
      this.onmessage = null;
      this.onerror = null;
      setTimeout(() => this.onerror?.(new Event("error")), 0);
    }
    close() { this.readyState = 2; }
    addEventListener() {}
    removeEventListener() {}
    dispatchEvent() { return true; }
  };
}
