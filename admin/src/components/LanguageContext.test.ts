import { describe, expect, it } from "vitest";
import { translate, translateText } from "./LanguageContext";

describe("admin translations", () => {
  it("translates shared navigation labels to Persian without changing English", () => {
    expect(translate("Dashboard", "fa")).toBe("داشبورد");
    expect(translate("Dashboard", "en")).toBe("Dashboard");
  });

  it("keeps unknown labels safe for incremental page translation", () => {
    expect(translate("New page label", "fa")).toBe("New page label");
  });

  it("translates labels embedded in dynamic global-time sentences", () => {
    expect(translateText("Global time range: Last 4 minutes", "fa")).toContain("بازه زمانی سراسری");
  });
});
