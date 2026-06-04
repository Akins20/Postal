import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";
import prettier from "eslint-config-prettier";

// eslint-config-next already bundles eslint-plugin-jsx-a11y, so its a11y rules
// are active without re-registering the plugin here (FRONTEND_PLAN §9.2).
const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  // Project-wide guardrails (FRONTEND_PLAN §9).
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      // Security: ban dangerouslySetInnerHTML outright (FRONTEND_PLAN §9.1).
      "no-restricted-syntax": [
        "error",
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message: "dangerouslySetInnerHTML is banned (FRONTEND_PLAN §9.1).",
        },
      ],
    },
  },

  // Layer boundaries (FRONTEND_PLAN §7): ui/ is presentational — no data/network.
  {
    files: ["src/ui/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/data/*", "@/features/*", "@/app/*", "@/api/*"],
              message: "ui/ must not import data/features/app/api (FRONTEND_PLAN §7).",
            },
          ],
        },
      ],
    },
  },
  // data/ is the network layer — no ui/feature imports.
  {
    files: ["src/data/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/ui/*", "@/features/*", "@/app/*"],
              message: "data/ must not import ui/features/app (FRONTEND_PLAN §7).",
            },
          ],
        },
      ],
    },
  },

  // Prettier last: disable formatting rules it owns.
  prettier,

  globalIgnores([
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    "src/api/schema.d.ts",
    "public/**",
    "coverage/**",
    "playwright-report/**",
  ]),
]);

export default eslintConfig;
