// ESLint for the mobile app: Expo's flat config plus the same layer
// boundaries as web (ui/ must not import data/features; data/ must not
// import ui/features).
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  { ignores: ["node_modules/*", ".expo/*", "dist/*", "src/api/schema.d.ts"] },
  {
    // Test files use require() inside jest.mock factories (hoisting rules).
    files: ["**/__tests__/**", "src/test/**"],
    rules: { "@typescript-eslint/no-require-imports": "off", "no-console": "off" },
  },
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
