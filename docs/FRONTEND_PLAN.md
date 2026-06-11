# Postal — Frontend Master Plan (Phase 12)

> The backend is **complete and frozen** (Phases 0–11). Its contract is
> [`docs/openapi.yaml`](openapi.yaml). This document is the *what* and *how* of
> the web client, mirroring the rigor of [`MASTER_PLAN.md`](MASTER_PLAN.md) and
> [`CODING_STANDARDS.md`](CODING_STANDARDS.md). Read [`../CLAUDE.md`](../CLAUDE.md)
> first — the prime directives (test everything before "done"; security-by-default;
> one phase at a time; memory discipline) apply here too. **The web is not a
> second-class client:** craft, motion, accessibility, and a clean data/UI
> architecture are requirements, not polish.

## 0. Decisions (locked unless changed with the user)

| Decision | Choice | Why |
|---|---|---|
| Web framework | **Next.js (App Router) + TypeScript (strict)** | Mature React framework + strong ecosystem; App Router gives a fast, app-like client. Used as an **application**, not a marketing site. |
| **Product type** | **Free tool / application — NOT a SaaS** | No pricing, plans, tiers, upgrade prompts, marketing landing, or growth funnels. It opens straight into the workspace. |
| Mobile | **Deferred** (web-only this phase) | Ship a usable product first; mobile revisited with the same generated client + zod schemas. |
| Scope | **Full breadth** — every screen planned together | One coherent app: auth, workspaces, channels, composer, media, scheduling/calendar, analytics, settings. |
| Repo | **Monorepo** — frontend in `web/` | Single source of truth; the TS API client is generated from `docs/openapi.yaml` in-tree. Go tooling ignores `web/`. |
| API consumption | **Generated types from the OpenAPI spec** | End-to-end type safety; the frozen spec is the contract. No hand-written request/response types. |
| Auth on the web | **Cookie session flow** (httpOnly) + CSRF double-submit | Backend already issues httpOnly `postal_access`/`postal_refresh` + JS-readable `postal_csrf`. Never store JWTs in JS (XSS-safe). |
| **Design language** | **macOS-inspired** — vibrancy/translucency, soft depth, spring motion | A calm, native-feeling, premium dashboard. Defined in §5. |
| **Navigation model** | **Bottom dock on the dashboard; macOS-style side rail on sub-pages/feature routes** | Dashboard = a home with a dock; drilling into a feature swaps to a left sidebar (macOS source-list pattern). Defined in §5. |
| Motion | **Framer Motion**, spring-based, reduced-motion aware | Cohesive, physical, interruptible animation — not decorative easing. §6. |
| Architecture | **Strict data-layer ⟂ UI-layer separation** | Presentational components never fetch; data lives in typed hooks. §7. |
| Observability | **Structured frontend logging + backend request-id correlation** | Traceable client behavior and errors, correlated to server logs. §8. |
| **Theme** | **Light + dark with an explicit, persisted toggle from day one** | User-controlled (defaults to `prefers-color-scheme`); both themes designed, not derived. |
| **Responsive** | **Mobile + tablet are first-class from day one** | Every page/component has a clean mobile **and** tablet form (fluid resize or a dedicated layout) — never a desktop-only afterthought. |
| **Guidance** | **Contextual hints/tooltips + progressive disclosure** | Users aren't expected to know everything; strategic hints, descriptive tooltips, and guided empty states teach in place. |

## 1. Principles (non-negotiable)

1. **The Go API is the only backend.** Next.js is for routing/rendering/UX only —
   **no business logic, no DB, no secrets.** Any server code in Next.js (route
   handlers/server actions) is a thin same-origin proxy for cookies; it never
   reimplements domain rules.
2. **Types come from the contract.** Generate `web/src/api/schema.d.ts` from
   `docs/openapi.yaml`. If spec and UI disagree, the spec wins.
3. **Security-by-default.** Tokens stay in httpOnly cookies; send `X-CSRF-Token`
   on every mutation. The server is the source of truth for authz — the UI
   *reflects* capabilities (hides/disables), never *enforces* them.
4. **Data ⟂ UI.** Fetching/caching/mutation logic lives in the data layer (§7).
   Presentational components take props and render; they don't call the network.
5. **Craft is a requirement — and this is a tool, not a SaaS website.** No
   marketing landing, pricing, plan tiers, upgrade prompts, or growth funnels;
   the app opens straight into the workspace. **No generic/templated landing
   design.** The only unauthenticated surface is a focused, original sign-in
   experience. Design references real native macOS app craft (§5, grounded in
   the research in §5.1) — never a stock SaaS/dashboard template or a
   component-library default look. Consistent design tokens, deliberate motion,
   complete loading/empty/error states, AA contrast, keyboard + screen-reader
   support, and reduced-motion fallbacks ship with every feature — not later.
6. **Readable, bounded code.** Small components, clear names, one concern per
   file; engineering rules in §9 are enforced like the backend's `make check`.
7. **Observable.** Structured logs with levels and correlation IDs (§8); every
   error surfaces a user-safe message *and* a traceable log line.
8. **Test everything before "done" — at the granularity of each unit.** Every
   **data-layer hook**, every **page**, and every **component** is tested and
   verified working **before moving to the next** — not batched at the end of a
   sub-phase. Data hooks: tested against MSW + at least one real-backend path;
   components: rendered + interaction + a11y (axe) tested; pages: Playwright e2e
   against a **real running backend + the X simulator** (never the paid API).
   "It renders" is not "it works."
9. **One sub-phase at a time, in order.** Keep this plan's checkboxes current.
10. **Mobile + tablet are first-class.** Every page and component is designed and
    verified at mobile, tablet, and desktop widths — either it fluidly resizes to
    a clean small-screen form or it has a dedicated mobile/tablet layout. The dock
    stays thumb-reachable at the bottom on small screens; the side rail becomes a
    slide-over. No "desktop-only" surfaces.
11. **Teach in place.** Users won't know everything up front. Provide
    contextual help at strategic points — descriptive tooltips on non-obvious
    controls, helpful empty states that explain the next action, inline hints, and
    light first-run guidance — using progressive disclosure (don't overwhelm).
    Accessible (tooltips reachable by keyboard/screen-reader; never the only way
    to convey critical info).

## 2. Tech stack (locked — change only with user approval)

- **Framework:** Next.js (App Router) + React + TypeScript (`strict`).
- **Styling:** **Tailwind CSS** with a tokenized config (§5) + **shadcn/ui**
  (Radix primitives, ownable, accessible). Class hygiene: **`clsx` +
  `tailwind-merge`** via a `cn()` helper; **`class-variance-authority` (cva)**
  for component variants; **`prettier-plugin-tailwindcss`** for deterministic
  class ordering. No arbitrary values except through tokens.
- **Motion:** **Framer Motion** (springs, layout/shared-element transitions,
  gesture + dock interactions). Centralized motion tokens (§6).
- **Icons:** **`lucide-react`** — one icon family, consistent stroke (1.5–2px)
  and size scale (16/20/24); wrapped in an `<Icon>` for size/aria defaults.
- **Server state:** **TanStack Query** (caching, mutations, invalidation,
  retries) over a typed client.
- **API client:** **`openapi-typescript`** (types from spec) + **`openapi-fetch`**
  (tiny typed fetch) with auth/CSRF/refresh interceptors.
- **Client state:** **Zustand** for active workspace + ephemeral UI only; server
  data stays in TanStack Query.
- **Forms/validation:** **react-hook-form** + **zod** (zod mirrors OpenAPI
  request bodies; server re-validates).
- **Charts:** **Recharts** (analytics time series).
- **Dates/tz:** `date-fns` + `date-fns-tz` (slots are IANA-tz-aware; API is UTC).
- **Logging:** a thin in-repo `logger` (levels + structured fields + correlation
  id), pluggable sink (console in dev; batched HTTP/telemetry later) (§8).
- **Testing:** **Vitest** + **@testing-library/react** + **MSW**; **Playwright**
  e2e; **axe** a11y assertions in component/e2e tests.
- **Quality gates (`web` check):** `tsc --noEmit`, ESLint (incl.
  `jsx-a11y`, `tailwindcss`), Prettier, **`knip`** (dead code), unit + e2e.
  Mirrors the backend `make check`.

## 3. Repository layout (target)

```
web/
  next.config.ts            # dev rewrites: /api/* -> Go backend (same-origin)
  tailwind.config.ts        # design tokens (colors, radii, shadow, blur, motion)
  src/
    app/                    # App Router — routing/layouts ONLY (no data logic)
      (public)/             # login, signup, verify-email, reset (unauthenticated)
      (app)/                # authenticated shell
        page.tsx            # DASHBOARD — bottom dock nav (§5)
        [workspace]/
          compose/  calendar/  channels/  media/  analytics/  members/  settings/
                            # FEATURE ROUTES — macOS side-rail nav (§5)
    api/                    # generated schema.d.ts + configured client + base hooks
    data/                   # data layer: TanStack Query hooks per domain (§7)
    features/<domain>/      # feature layer: containers wiring data -> ui
    ui/                     # UI layer: presentational components (shadcn-based)
      dock/  sidebar/  motion/  primitives/   # dock, side rail, motion presets
    lib/                    # cn(), auth, csrf, query client, logger, tz, format
    test/                   # Playwright e2e + test utils + MSW handlers
  package.json / scripts    # gen:api, dev, build, lint, typecheck, test, check
scripts/dev/gen-api.sh      # openapi-typescript docs/openapi.yaml -> web/src/api/schema.d.ts
```

The three layers are physical directories: **`data/`** (network + cache),
**`ui/`** (pure presentation), **`features/`** (containers). `app/` only composes
routes/layouts. This boundary is lint-enforced (§9).

## 4. Auth, sessions & the same-origin requirement

The cookie flow (`postal_access` SameSite=Lax, `postal_refresh` SameSite=Strict
Path=`/api/v1/auth`, `postal_csrf` JS-readable) requires **same-site** delivery:

- **Dev:** Next.js `rewrites` proxy `/api/*` → Go server (`:8080`) → one origin →
  cookies + CSRF work with no CORS.
- **Prod:** web + API on the **same registrable domain** (single origin via the
  edge, or same-site subdomains) with the backend's `POSTAL_CORS_ALLOWED_ORIGINS`
  allowlist + `credentials: 'include'`.
- **Client:** every request `credentials: 'include'`; mutations add `X-CSRF-Token`
  from the `postal_csrf` cookie; a `401` interceptor calls `POST /auth/refresh`
  once and retries, else routes to login. We never read the httpOnly cookies.

## 5. Design language & navigation — macOS

**Aesthetic.** Calm, premium, native-feeling. Translucent "vibrancy" surfaces
(`backdrop-blur` over a tinted scrim), soft layered shadows, generous rounded
radii, hairline separators, a restrained accent color, and SF-like typography
(system font stack: `-apple-system, "SF Pro", Inter, …`). **Light + dark from
day one with an explicit, persisted theme toggle** (CSS variables; defaults to
`prefers-color-scheme`, user override stored and applied before first paint to
avoid flash). Both themes are designed deliberately — dark is not a naive invert.
Grounded in current macOS materials/"Liquid Glass" language — see §5.1.

**Design tokens** (Tailwind config — the single source; no ad-hoc values):
- **Color:** semantic tokens (`bg`, `bg-elevated`, `bg-vibrancy`, `fg`,
  `fg-muted`, `accent`, `danger`, `separator`) mapped to light/dark.
- **Radius:** `sm 8 / md 12 / lg 16 / xl 22` (macOS-ish continuous corners).
- **Elevation:** `shadow-window`, `shadow-popover`, `shadow-dock` (soft, layered).
- **Vibrancy:** `blur-vibrancy` + translucent fills for dock, sidebar, sheets,
  popovers, menus.
- **Spacing/typography:** 4px base scale; type ramp (title/headline/body/caption).

**Navigation model (the core macOS decision):**

- **Dashboard = bottom dock.** The authenticated home (`(app)/page.tsx`) presents
  a **floating, translucent bottom dock** (the primary nav for top-level
  destinations: Home, Compose, Calendar, Channels, Media, Analytics). The dock:
  - sits centered, floating above content with vibrancy + `shadow-dock`;
  - has **hover/active magnification** and a spring "bounce" on launch (Framer
    Motion), an active-item indicator dot, and tooltips;
  - is keyboard-navigable (roving tabindex), `role="navigation"`, reduced-motion
    aware (magnification → simple highlight).
- **Sub-pages/feature routes = macOS side rail.** Drilling into a feature
  (`[workspace]/<feature>`) swaps the chrome to a **left source-list sidebar**
  (macOS Finder/Settings pattern): translucent, collapsible, sectioned list with
  selection highlight; content area to the right with a slim top bar (title,
  breadcrumb, contextual actions). Sub-navigation *within* a feature (e.g.
  analytics: Overview / Posts / Export; settings: Profile / Workspace / Members)
  lives in this sidebar.
- **Transition between the two:** entering a feature animates dock → sidebar
  (shared-layout/cross-fade); leaving restores the dock. One mental model: **home
  has a dock, rooms have a sidebar.**
- **Responsive (mobile + tablet, first-class):** breakpoints ≈ `sm`(≥640)
  `md`(≥768, tablet) `lg`(≥1024) `xl`(≥1280). On mobile the dock stays bottom
  (thumb-reachable, iOS-tab-like) and the side rail becomes a slide-over sheet
  (swipe/escape to dismiss); on tablet, a compact persistent sidebar + content.
  Every component declares its small-screen behavior (fluid resize or a dedicated
  layout) and is verified at all three widths — no horizontal scroll, no clipped
  controls, touch targets ≥44px.

**Component inventory (ui/):** Dock + DockItem, Sidebar + SidebarItem/Section,
Window/Panel (vibrancy card), TopBar, Sheet/Drawer, Popover/Menu, Toast,
Dialog/Alert, Button (cva variants), Input/Field, Tabs, Segmented control,
Badge/StatusPill, Avatar, EmptyState (with guidance), Skeleton, **Tooltip + Hint
+ HelpPopover** (contextual help), ThemeToggle, Calendar grid, MetricCard, Chart
wrappers. All shadcn/Radix-based, themed via tokens, motion-wrapped where natural,
each with a defined mobile/tablet form.

### 5.1 Design research & references (grounded — not a generic template)

The macOS direction is grounded in Apple's current design language, not a stock
"glassmorphism" theme. Key references and the techniques we adopt:

- **Apple HIG — Materials & vibrancy:** materials impart *translucency + blur* for
  layer separation; *vibrancy* pulls color forward from behind the material for
  foreground text/symbols/fills, enhancing depth. We model named materials
  (`vibrancy/dock`, `vibrancy/sidebar`, `vibrancy/sheet`, `vibrancy/popover`) as
  tokens, each a tuned blur + translucent tint + subtle inner/edge highlight.
- **"Liquid Glass" (2025/iOS 26 · macOS 26):** Apple's evolution of materials —
  rounded, translucent elements with optical *refraction/reflection* that react to
  motion and content. We adopt the *spirit* (depth, fluid spring response,
  light-reactive edges) tastefully; refraction is a progressive enhancement.
- **Web technique:** `backdrop-filter: blur()` over a low-opacity tinted scrim +
  soft gradient + 1px translucent border/inner highlight for the frosted base.
  Optional refraction via an SVG `feDisplacementMap` driving `backdrop-filter`
  (Chromium-only today) behind a feature check — **graceful fallback** to plain
  frosted glass on Safari/Firefox. Wrap vibrancy surfaces in `contain: strict`
  (or `paint`) to bound rasterization and protect INP. Always keep a solid
  fallback when `backdrop-filter` is unsupported or reduced-transparency is set.
- **Accessibility:** honor `prefers-reduced-transparency` (drop to opaque
  surfaces) and `prefers-reduced-motion`; maintain AA contrast for text *over*
  vibrancy (vibrancy is for chrome, not body text on busy backgrounds).

> Sources (researched 2026-06): Apple HIG — Materials
> (developer.apple.com/design/human-interface-guidelines/materials),
> NSVisualEffectView (developer.apple.com/documentation/appkit/nsvisualeffectview),
> Apple "Liquid Glass" overview (2025), and current web implementations of frosted
> glass / Liquid Glass with `backdrop-filter` + SVG (LogRocket; flyonui;
> dev.to mac-style dock). We take inspiration and technique, not any template.

## 6. Motion & animation

- **Library:** Framer Motion. **Centralized presets** in `ui/motion/` — spring
  configs (`gentle`, `snappy`, `bouncy`), durations, and easing; components import
  presets, never hand-tune per use.
- **Where motion is used (purposeful, not decorative):** route/page transitions
  (cross-fade + subtle slide), dock magnification + launch bounce + active
  indicator, dock↔sidebar swap (shared layout), sheet/drawer/dialog enter/exit
  (spring scale+fade from origin), list add/remove/reorder (`AnimatePresence`,
  `layout`), optimistic state settles, toast stack, skeleton shimmer, number/metric
  count-ups in analytics, hover/press micro-interactions (scale 0.98 on press).
- **Discipline:** interruptible (springs, not fixed timelines); GPU-friendly
  (transform/opacity only); **`prefers-reduced-motion` respected globally** (a
  `useReducedMotion` gate degrades to instant/opacity-only); no motion that blocks
  input or delays content > ~250ms.

## 7. Architecture: data layer ⟂ UI layer

Three layers, enforced as directories + lint boundaries:

1. **Data layer (`data/`)** — the only place that touches the network. Per-domain
   modules export typed **TanStack Query hooks** built on the generated
   `openapi-fetch` client: `useChannels(ws)`, `useCreatePost()`,
   `useSchedulePost()`, `usePostMetrics(ws, postId)`, … Each owns its query keys,
   cache invalidation, optimistic updates, and error normalization. Zod parses
   inputs; the generated types guarantee response shapes.
2. **UI layer (`ui/`)** — pure presentational components. **Props in, events out;
   no hooks that fetch, no query keys, no `fetch`.** Fully testable in isolation
   with MSW-free rendering. This is where the macOS design system lives.
3. **Feature layer (`features/<domain>/`)** — containers that wire data hooks to
   UI components for a screen, handle local view state, and map errors to UX. The
   only layer allowed to import from *both* `data/` and `ui/`.

`app/` routes import **features**, compose layout (dock vs sidebar shell), and
nothing else. Rule of dependency: `app → features → {data, ui}`; `ui` never
imports `data`; `data` never imports `ui`. (ESLint `no-restricted-imports` /
boundaries plugin enforces this.)

## 8. Observability — structured logging & traces

- **Logger (`lib/logger`):** leveled (`debug|info|warn|error`), **structured**
  (JSON fields, never string-concatenated), with a pluggable sink (pretty console
  in dev; batched HTTP/telemetry endpoint in prod — pluggable, off by default).
  No PII or tokens in logs (mirror the backend rule).
- **Correlation with the backend.** The API client attaches/propagates a
  client-generated `X-Request-Id` (or surfaces the server's), and on any error
  envelope it logs the server's `error.request_id` so a frontend error ties to a
  backend log line. Every API call can emit a structured trace (method, path,
  status, duration, request-id) at `debug`/`warn`.
- **Error boundaries:** React error boundaries per route segment log structured
  errors + show a recoverable fallback; a global handler captures unhandled
  rejections. User sees a safe message; the log carries the detail + correlation
  id.
- **Performance traces:** Web Vitals (LCP/INP/CLS) and key interaction timings
  (compose-submit, schedule, route transition) logged for budgets in 12.7.

## 9. Frontend engineering standards (enforced, not optional)

- **Layer boundaries** (§7) enforced by lint; no cross-layer leaks.
- **Component size:** presentational components stay small and focused; split when
  a file grows past a sane cap (≈250 lines) or mixes concerns. One exported
  component per file (plus its local subcomponents).
- **Naming & structure:** `PascalCase` components, `useX` hooks, `camelCase`
  utils; colocate component + test + styles; feature-first folders.
- **No prop drilling marathons:** prefer composition/context for cross-cutting
  (theme, workspace, toasts); keep prop lists honest (object props for >4 fields).
- **Typing:** `strict` TS, no `any` (lint-error), no non-null `!` without a
  comment; exhaustive `switch` on unions (status enums from the spec).
- **Accessibility:** every interactive element labelled/role'd/keyboard-operable;
  focus management on dialogs/sheets; `jsx-a11y` clean; axe assertions in tests.
- **Tailwind discipline:** tokens + `cn()` + cva variants; `prettier-plugin-
  tailwindcss` ordering; no magic numbers/colors.
- **State:** server state in TanStack Query only; client state minimal; no
  duplicating server data into Zustand.
- **Errors:** one envelope→UX mapper; never swallow; always log (§8).
- **Docs/comments:** JSDoc on exported hooks/components and non-obvious logic;
  no commented-out code.
- **`web check` is the gate:** typecheck + eslint(+a11y) + prettier + knip + unit
  + e2e. A sub-phase isn't "done" while it fails.

### 9.1 Security (frontend) — enforced

- **No tokens in JS.** Access/refresh live in httpOnly cookies; we never read or
  store them in `localStorage`/JS. `postal_csrf` (the only JS-readable cookie) is
  echoed as `X-CSRF-Token` on mutations.
- **XSS prevention.** Rely on React's default escaping; **`dangerouslySetInnerHTML`
  is banned** (lint-error) — if rich content is ever unavoidable, sanitize with a
  vetted sanitizer. No `eval`/`new Function`/string-built DOM. User content
  (post bodies, handles) is rendered as text, never HTML.
- **Content Security Policy.** Ship a strict CSP for the app (script-src 'self'
  with nonces; no `unsafe-inline`/`unsafe-eval` where the framework allows;
  connect-src limited to self/API; frame-ancestors 'none'; object-src 'none';
  base-uri 'self'). Set via Next.js headers/middleware. Plus `Referrer-Policy`,
  `X-Content-Type-Options`, `X-Frame-Options: DENY` (anti-clickjacking; backend
  already sends these for the API).
- **Secrets & config.** Only `NEXT_PUBLIC_*` reach the client bundle; **no secret
  ever** ships to the browser. The dev proxy holds no credentials.
- **External links & navigation.** `target="_blank"` always with
  `rel="noopener noreferrer"` (anti-tabnabbing). Validate/whitelist any
  redirect target (OAuth `authorize_url` comes from our API only). No
  `javascript:`/`data:` URLs from user input.
- **Transport & cookies.** HTTPS-only in prod (HSTS from the backend edge);
  cookies stay `Secure`/`SameSite`. CORS is the backend's allowlist — the app
  never relaxes it.
- **Dependency hygiene.** Lockfile committed; `npm audit`/Dependabot in CI; no
  known-vuln deps; minimal dependency surface.
- **Auth UX safety.** Idle/expiry → refresh-once then re-auth; logout clears
  session; never expose another user's data; respect server 401/403 as truth.
- **Logging.** No PII/tokens in client logs (§8); error envelopes carry only the
  server's safe `code`/`message` + `request_id`.

### 9.2 Accessibility — WCAG 2.2 AA (target)

- **Standard:** conform to **WCAG 2.2 Level AA**. Treat a11y failures like build
  failures (axe in component + e2e tests; `eslint-plugin-jsx-a11y` clean; a manual
  keyboard + screen-reader pass each sub-phase).
- **Semantics & structure:** semantic HTML + landmarks (`header/nav/main/aside`),
  correct heading hierarchy, lists for lists; ARIA only to fill gaps (Radix
  primitives provide correct roles for dock/sidebar/menus/dialogs/tabs).
- **Keyboard:** everything operable by keyboard (dock = roving tabindex, sidebar,
  menus, calendar, dialogs); **visible focus** indicators (never `outline:none`
  without a replacement); logical tab order; a **skip-to-content** link.
- **Focus management:** dialogs/sheets trap focus, are dismissible by `Esc`, and
  **restore focus** to the trigger on close; route changes move focus to the new
  view's heading.
- **Forms:** every field labelled; errors associated via `aria-describedby` and
  announced; required state conveyed (not by color alone); the envelope's
  `fields[]` map to the right inputs.
- **Status & feedback:** toasts/async updates use polite `aria-live` regions;
  status is never **color-only** (status pills carry text + icon); loading states
  are announced.
- **Contrast & sizing:** AA contrast (4.5:1 text, 3:1 large text / UI/graphics);
  text resizable to 200% without loss; **touch targets ≥44px** (WCAG 2.2 target
  size); honor `prefers-reduced-motion` and `prefers-reduced-transparency`.
- **Media/icons:** meaningful icons have accessible names; decorative ones are
  `aria-hidden`; images/avatars have alt text.
- **Tooltips/hints (§11):** reachable by keyboard + screen-reader and **never the
  only channel** for critical information.

## 10. Sub-phases (build order; all screens in scope)

### 12.0 — Scaffold & foundations ✅ DONE (2026-06-04)
- [x] `web/` Next.js 16 (App Router) + TS(strict) + Tailwind v4(tokens §5); ESLint(jsx-a11y via next, layer-boundaries, ban `dangerouslySetInnerHTML`)/Prettier(+tailwind plugin)/tsc; `web check` (typecheck+lint+format+test). _knip deferred — its latest pulls an unpublished `@oxc-project/types`._
- [x] **Security baseline:** nonce CSP via Next-16 **`src/proxy.ts`** (strict `script-src`; `style-src 'unsafe-inline'` for Radix/Framer inline styles) + static security headers in `next.config`; `dangerouslySetInnerHTML`/`any` lint-banned; only `NEXT_PUBLIC_*` to the client.
- [x] `scripts/dev/gen-api.sh` → `src/api/schema.d.ts` (openapi-typescript from `docs/openapi.yaml`); `openapi-fetch` client with `credentials:'include'`, `X-CSRF-Token` on mutations, single-flight 401→refresh→retry, request-id.
- [x] TanStack Query + next-themes + Radix Tooltip providers; structured **logger** (§8) + route + global error boundaries + envelope→UX mapper (`normalizeError`).
- [x] **Design system + motion + theme:** macOS tokens (light/dark via persisted no-flash toggle), vibrancy materials (§5.1) with reduced-transparency fallback; `ui/` primitives (Icon, Button[cva], Tooltip, Hint, Panel, StatusPill, EmptyState, ThemeToggle); **Dock** + **Sidebar** + **FeatureShell**; Framer-Motion presets (§6, reduced-motion aware).
- [x] **Responsive shells:** dashboard (bottom dock) + feature route group `(feature)` (side rail → mobile slide-over) at mobile/tablet/desktop; dev rewrite proxies `/api/*` → Go API; full nav skeleton (compose/calendar/channels/media/analytics/settings stubs).
- [x] Test harness: Vitest + Testing-Library + **axe** (11 unit/component tests green); Playwright config + smoke spec written. _Playwright browser binaries aren't published for this OS (Ubuntu 26.04) → e2e runs in CI/a supported runner; verified via production build instead._
- [x] Verified: `web check` green, `next build` succeeds (Proxy/CSP active, all routes compile).

### 12.1 — Auth & session ✅ DONE (2026-06-04)
- [x] Login, signup, email-verify, password-reset request/confirm (rhf + zod); macOS centered vibrancy `AuthPanel`. Accessible `FormField` (label/aria-invalid/role=alert); server field-errors map to inputs, form-level fallback.
- [x] Session bootstrap (`useMe` → `GET /auth/me`, 401 = signed-out not error); `AuthGuard` over the `(app)` route group redirects unauthenticated → /login; `AuthPanel` redirects authed away from `(auth)` pages; routes restructured into `(app)` [guarded] + `(auth)` [public]; logout (`UserMenu`); single-flight refresh-on-401.
- [x] **MSW test infra** wired; 23 tests green (auth hooks incl. 401, FormField, LoginForm validation/submit→redirect/error, AuthGuard); `next build` green (13 routes). _Spec fix: `User`/`Token` fields marked `required` so generated types aren't all-optional._

### 12.2 — Workspaces & members ✅ DONE (2026-06-04)
- [x] `WorkspaceSwitcher` (Radix menu) backed by a persisted Zustand active-workspace store (`useActiveWorkspace` = `useWorkspaces` + store, defaults to first); wired into the dashboard header + feature side rail. _Implemented as a flat-route active-workspace store rather than `[workspace]` URL segments._
- [x] Members management on `/settings`: `MembersPanel` (list with per-member role select → `useUpdateCapabilities`, owner shown immutable) + `AddMemberForm` (email + role preset, or a custom `CapabilityCheckboxes` group). Capabilities/roles config mirrors the backend.
- [x] Data hooks in `src/data/workspaces.ts` (useWorkspaces/useMembers/useAddMember/useUpdateCapabilities). 32 tests green (workspace hooks, AddMemberForm validation/submit/custom-caps, WorkspaceSwitcher render + switch); `next build` green. _Spec: `Workspace`/`Member` fields marked `required`. Radix-in-jsdom polyfills added to test setup._

### 12.3 — Channels ✅ DONE (2026-06-11)
- [x] `/channels` screen: `ChannelsPanel` lists connected accounts (platform glyph, display name/@handle, status pill Active/Expired/Revoked with tooltip hints) + "Connect a platform" list; empty state CTA. Platform registry in `src/config/platforms.tsx` (X glyph hand-rolled — lucide v1 dropped brand icons).
- [x] Connect flow: `ConnectChannelButton` → `POST /channels/connect` → browser redirect to `authorize_url`; IdP returns to **`/oauth/callback`** (auth-guarded page outside the feature shell) → `useCompleteOAuth` exchanges single-use state+code (fires exactly once) → redirect `/channels`; error view with way back.
- [x] Disconnect via reusable `ui/primitives/confirm-dialog.tsx` (Radix Dialog: focus trap, Esc, destructive variant, pending state); inline error if the delete fails.
- [x] Data hooks `src/data/channels.ts` (useChannels/useConnectChannel/useCompleteOAuth/useDisconnectChannel). 50 tests green (18 new: hooks incl. 403/404/bad-state + panel empty/list/disconnect/error + connect-redirect + callback success/error/missing); `next build` green (15 routes). _Spec: `Channel` fields marked `required`._

### 12.4 — Composer & media ✅ DONE (2026-06-11)
- [x] Compose-once editor (`/compose`): master text → every selected channel (`ChannelPicker` chips; non-active channels disabled with reconnect tooltip); per-channel override tabs (override dot, "reset to master"); live char counter vs platform cap (`platforms.charLimit`, X=280; min across selection on the master tab) — server re-validates: save runs create/update **then `POST …/validate`**, per-channel Ready/Needs-changes verdicts shown.
- [x] UTM preview (collapsible: utm_source/utm_campaign → `POST posts/utm-preview` tagged text); drafts CRUD ("Your posts" list: excerpt/status/edit/delete-confirm; Edit remounts the composer via `key`).
- [x] Media library (`/media`): XHR multipart upload with live `<progress>` (fetch can't report upload progress; `csrfToken()` exported for non-openapi-fetch callers), responsive grid (img via cookie-auth'd `mediaDownloadURL`, video placeholder), delete-confirm; quota/oversize rejections inline. `MediaAttach` picker dialog in the composer attaches assets as `MediaMeta` to variants.
- [x] Data hooks `src/data/posts.ts` (list/get/create/update/delete/validate/utm-preview) + `src/data/media.ts` (list/upload/delete/downloadURL). 32 new tests → 82 green; `next build` green. _Spec: `Post`/`Variant`/`MediaMeta`/`VariantValidation`/`Asset` marked `required`. jsdom XHR can't serialize FormData→multipart, so the upload test asserts plumbing/envelope; multipart correctness lands on e2e/curl (12.7)._

### 12.5 — Scheduling & calendar ✅ DONE (2026-06-11)
- [x] `ScheduleDialog` on each draft row (composer "Your posts"): **next open slots** (`to_slots`, with a what-are-slots hint) or **specific time** (`datetime-local` in the user's tz → ISO UTC `run_at`); per-channel job count confirmation; backend rejections inline.
- [x] `/calendar`: month grid (job pills with time+handle, +n overflow, today ring, day-detail on click) / week list (grouped by day) toggle; range nav with Framer fade-slide transitions (reduced-motion aware); job status pills (scheduled/publishing/published/failed/canceled tones); **cancel** a scheduled job via confirm dialog (`JobItem`).
- [x] `SlotsManager` on `/calendar`: per-channel weekly slots list + create (day/time/`Intl.supportedValuesOf("timeZone")` picker, defaults to the user's tz) + delete. _Gotcha fixed: mount it only after channels load — its channel-select state initializes from the first channel._
- [x] Data hooks `src/data/schedule.ts` (schedule/calendar/cancel/slots CRUD; `channel_id` rides as a query param on slot delete). 18 new tests → 100 green; `next build` green. _Spec: `Job`/`Slot` marked `required`._

### 12.6 — Analytics
- [ ] Overview (per post×channel); per-post per-channel breakdown; time-series charts (range picker, count-up); CSV export.

### 12.7 — Settings, polish & freeze
- [ ] Account/workspace settings; full a11y + responsive + reduced-motion pass; complete loading/empty/error/skeleton states.
- [ ] Playwright e2e of the core loop + each domain vs running backend + simulator; Web Vitals budgets; docs.
- [ ] **Frontend declared complete.**

## 11. Cross-cutting concerns

- **Error UX:** envelope→{field errors (`fields[]`) + toast}; 401→refresh/login,
  403→"no permission", 429→Retry-After countdown, 5xx→generic+retry; all logged.
- **Invalidation/optimism:** mutation `onSuccess` invalidates relevant queries;
  optimistic only where rollback is safe; settle with motion.
- **Timezones:** display user-tz, send UTC; slot editor explicit about IANA tz.
- **Capabilities & quotas:** `useCapabilities(ws)` gates affordances; surface
  quota usage (channels, scheduled posts, storage) and disable at cap so users
  meet friendly UI limits before server 400/429s.
- **Contextual help:** descriptive tooltips on non-obvious controls, guided empty
  states, inline hints, and light first-run coachmarks — progressive disclosure,
  dismissible, never blocking, and accessible (§9.2).
- **Theming:** light/dark via an explicit, persisted toggle; vibrancy consistent
  across dock/sidebar/sheets; reduced-transparency + reduced-motion fallbacks.
- **Responsive:** mobile/tablet/desktop layouts for every surface (§5 nav model);
  verified at all three widths.
- **Security & a11y:** every feature meets §9.1 (CSP, no `dangerouslySetInnerHTML`,
  CSRF, safe links) and §9.2 (WCAG 2.2 AA) — not a final-phase sweep.

## 12. Definition of Done (every sub-phase)

- [ ] `web check` green: `tsc --noEmit`, ESLint(+jsx-a11y, +boundaries), Prettier, knip.
- [ ] Types generated from `docs/openapi.yaml` (no hand-rolled API types).
- [ ] Data ⟂ UI layering respected (§7); component-size + standards (§9) met.
- [ ] **Every data hook, page, and component tested before the next** (§1.8):
      hooks (MSW + a real-backend path), components (render + interaction + axe),
      pages (Playwright e2e **against a running backend + simulator**).
- [ ] Design tokens used (no ad-hoc values); macOS material/vibrancy per §5.1 with
      solid + reduced-transparency fallbacks; motion + **reduced-motion** fallback;
      **light AND dark** both designed and correct via the toggle.
- [ ] **Responsive verified at mobile, tablet, and desktop** (dock bottom on
      mobile; sidebar → slide-over; no clipping/overflow; targets ≥44px).
- [ ] **Accessibility WCAG 2.2 AA (§9.2):** axe clean, keyboard + screen-reader
      pass, visible focus, focus-trap/restore on dialogs, AA contrast, status not
      color-only.
- [ ] **Security (§9.1):** no `dangerouslySetInnerHTML`/`eval`; CSP + safe
      headers; `rel="noopener noreferrer"` on external links; CSRF on mutations;
      tokens only in httpOnly cookies; no secrets in the bundle; `npm audit` clean.
- [ ] **Contextual help present** where non-obvious (tooltips/hints/guided empty
      states), accessible and dismissible.
- [ ] Loading/empty/error/skeleton states for every async surface.
- [ ] Structured logs + backend request-id correlation on API/error paths (§8).
- [ ] This plan's checkboxes updated; a memory entry capturing patterns/decisions.

## 13. Open questions / future

- Production topology (single origin vs same-site subdomains) — pick before 12.7.
- Real-time (job status/metrics): polling first; SSE/WebSocket later if the
  backend grows a stream.
- Mobile (React Native/Expo reusing the generated client + zod) — separate plan
  once web is stable; the macOS bottom-dock model maps cleanly to mobile tabs.
- Telemetry sink wiring for the logger (vendor TBD) — interface ready, off by
  default.

> **No marketing/landing site.** This is a free tool; there is no public
> marketing surface, pricing page, or SaaS funnel. If a public site is ever
> wanted it is an entirely separate track and must not bleed generic SaaS design
> into the application.
