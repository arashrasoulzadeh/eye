import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    "public/cdn-eye.min.js",
    "public/cdn-eye.min.js.map",
  ]),
  { rules: {
    // These effects intentionally synchronize async API subscriptions with
    // component state; the React compiler rule treats the async call site as
    // a synchronous state update even though the network callback is later.
    "react-hooks/set-state-in-effect": "off",
    // Date.now() in an explicit user action handler is intentionally sampled
    // at click time, not during render.
    "react-hooks/purity": "off",
    "@typescript-eslint/no-explicit-any": "off",
  } },
]);

export default eslintConfig;
