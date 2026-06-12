// ESLint for the mobile app: Expo's flat config plus the same layer
// boundaries as web (ui/ must not import data/features; data/ must not
// import ui/features).
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  { ignores: ["node_modules/*", ".expo/*", "src/api/schema.d.ts"] },
  {
    files: ["src/ui/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        { patterns: [{ group: ["@/data/*", "@/features/*", "@/app/*"], message: "ui/ stays presentational." }] },
      ],
    },
  },
  {
    files: ["src/data/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        { patterns: [{ group: ["@/ui/*", "@/features/*", "@/app/*"], message: "data/ is the network layer only." }] },
      ],
    },
  },
]);
