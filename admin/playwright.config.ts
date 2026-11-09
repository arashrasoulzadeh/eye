import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  fullyParallel: false,
  retries: 0,
  reporter: "list",
  use: {
    baseURL: "http://localhost:3100",
    headless: true,
  },
  webServer: [
    {
      command: "rm -f /tmp/eye-e2e.db && cd .. && go run ./server/cmd/server --env=.env.e2e",
      url: "http://localhost:8099/metrics",
      reuseExistingServer: false,
      timeout: 60_000,
    },
    {
      command: "npm run dev -- --webpack --port 3100",
      url: "http://localhost:3100/login",
      reuseExistingServer: false,
      timeout: 60_000,
      env: {
        BACKEND_URL: "http://localhost:8099",
      },
    },
  ],
});
