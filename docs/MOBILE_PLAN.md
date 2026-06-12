# Postal Mobile (Android-first) - Master Plan (Phase 15)

> Status: PLANNING (drafted 2026-06-12). Build starts only after the plan is
> approved. Web stays the reference implementation; the backend is FROZEN and
> already client-agnostic (Bearer JWT, JSON envelopes, OpenAPI).

## 0. Decisions (proposed; flag before changing)

| Decision | Choice | Why |
| --- | --- | --- |
| Framework | **React Native + Expo (TypeScript)** | Maximum pattern reuse from web: same language, same TanStack Query data-layer shape, same OpenAPI-generated types, same zustand stores. The team already has a proven web codebase to mirror; a Kotlin/Compose rewrite shares nothing. iOS comes nearly free later. |
| Styling | **NativeWind v4** (Tailwind for RN) + the same oklch token palette | The web design system ports as tokens, not screenshots. One `tokens.ts` source of truth. |
| Navigation | **Expo Router** (file-based, like Next) + native **bottom tab bar** | The dock's mobile-native counterpart is a bottom tab bar; Expo Router mirrors the `app/` routing mental model we already use. |
| API client | `openapi-fetch` + generated `schema.d.ts` from `docs/openapi.yaml` | Identical to web; regen script already exists. |
| Auth | **Bearer JWT** + refresh token in **Android Keystore** (expo-secure-store) | The cookie/CSRF dance is a browser concern; the backend's Bearer + body-refresh flow was built for exactly this client. Access token lives in memory only. |
| State/data | TanStack Query v5 + zustand | Copy the web's `data/` hooks almost verbatim. |
| Testing | Jest + React Native Testing Library + **MSW** (unit), **Maestro** (e2e on emulator against the local stack + simulators) | Same test-every-unit cadence as web; Maestro plays the Playwright role. |
| Distribution | EAS Build -> Google Play **internal testing** track first | Standard Expo pipeline; signing stays in EAS. |
| Repo layout | `mobile/` directory beside `web/` | Same repo, same OpenAPI source, same docs discipline. |

## 1. Principles (carried over from web)

- **Not a SaaS app**: no marketing, pricing pages, or upsell chrome.
- **Test every data hook and component before moving on**; failure paths included.
- **Security non-negotiable**: tokens in Keystore, never AsyncStorage; no
  secrets in the bundle; HTTPS only in production builds; certificate of the
  API origin pinned as a later hardening item.
- **Hints everywhere**: the same contextual tooltips/hints, adapted to
  long-press and info-icon affordances.
- Light/dark from day one, following the system with an in-app override.
- No em dashes in user-facing copy.

## 2. Design language: the same scheme, translated to Android

The web app is macOS-flavored; mobile keeps the *scheme* (tokens, materials,
motion, iconography) while using native interaction patterns:

- **Tokens**: port `globals.css` oklch palette to a shared `tokens.ts`
  (surface/elevated/fg/fg-muted/fg-subtle/accent/danger/success/warning/
  separator + light/dark variants). NativeWind consumes it; any future
  rebrand edits one file per client.
- **Dock -> bottom tab bar**: Home, Compose, Calendar, Channels, More.
  "More" sheet carries Media, Analytics, Wallet, Integrations, Settings
  (nine top-level items don't fit a phone tab bar honestly).
- **Vibrancy -> expo-blur** surfaces for the tab bar and sheets, with the
  same opaque fallback rule (reduced-transparency accessibility setting).
- **Cards**: same Panel idiom (rounded-xl, hairline border, layered
  elevation), same icon-chip page headers.
- **Motion**: react-native-reanimated springs tuned to the web presets
  (gentle/snappy/bouncy); reduced-motion honored via AccessibilityInfo.
- **Type**: Inter via expo-font, same ramp; tabular numerals for balances.
- **Icons**: lucide-react-native (same set as web) + the existing brand
  glyphs (X, Instagram camera, TikTok note) as react-native-svg components.

## 3. Architecture (mirrors web exactly)

```
mobile/
  app/                    # Expo Router routes (login, tabs, modals)
    (auth)/login|signup|reset
    (app)/(tabs)/index|compose|calendar|channels|more
    (app)/oauth-callback  # deep-link target
  src/
    api/                  # openapi-fetch client + generated schema.d.ts
    data/                 # TanStack Query hooks - ported from web/src/data
    features/<domain>/    # containers wiring data -> ui
    ui/                   # primitives: Panel, Button, StatusPill, Hint...
    stores/               # zustand (active workspace, theme override)
    lib/                  # tokens, format (atHandle/formatBytes), logger
```

Layer rules enforced with the same eslint `no-restricted-imports` boundaries.

## 4. The four genuinely mobile problems (and their answers)

1. **Auth/session**: login -> `{access_token, refresh_token}`; refresh token
   to Keystore, access in memory; single-flight refresh-on-401 interceptor
   identical in shape to the web's `apiFetch`. Logout revokes via body
   refresh-token. No cookies, no CSRF (Bearer clients are exempt by design).
2. **OAuth channel connect**: open `authorize_url` in **Chrome Custom Tabs**
   (expo-web-browser); the platform redirects to the registered redirect URI.
   Mobile registers `https://app.postal.example/oauth/callback` as an Android
   **App Link** so the same URI works for web AND app (dev: `postal://oauth/
   callback` custom scheme against the simulators, which accept any redirect).
   The app then calls `GET /channels/oauth/callback?state&code` exactly like
   the web page does. Backend change required: none (state is client-agnostic).
3. **Media upload**: expo-image-picker -> `FormData` with file URI ->
   existing multipart endpoint; progress via expo-file-system `uploadAsync`.
   Camera capture is a free add.
4. **Wallet & Google Play policy**: selling credits in-app likely triggers
   Play Billing rules (15-30% cut, or the alternative-billing programs).
   **v1 ships the wallet read-only**: balance, ledger, price tiers, plus a
   "top up from any browser at /wallet" note. Checkout stays on the web.
   Revisit Play Billing/alternative billing as its own later phase with the
   operator's business call. (This is the one place mobile intentionally
   diverges from web.)

## 5. Sub-phases (one at a time, verified before the next)

- [x] **15.0 Scaffold** DONE 2026-06-12: Expo SDK 56 (RN 0.85, React 19.2,
  TS 6, react-compiler on) in `mobile/`; oklch palette ported to
  `src/lib/tokens.ts` (hex; names match web 1:1) with a system+override
  theme store; Panel/Button/StatusPill primitives; five-tab Expo Router
  shell (Home/Compose/Calendar/Channels/More) with lucide icons; typed
  `openapi-fetch` client (Bearer flow, 10.0.2.2 dev origin, request-id) +
  `gen-api-mobile.sh`; eslint flat config with the web's layer boundaries;
  Jest (jest-expo) + RNTL harness - 6 tests green; typecheck/lint green;
  `expo export --platform android` bundles clean (5.7MB hbc).
  _Decision deltas from this plan, both recorded for review: JS `Tabs`
  instead of unstable_ NativeTabs (lucide + blur styling need custom
  icons); tokens.ts + StyleSheet instead of NativeWind for now (RN 0.85 +
  React 19.2 is too new to bet on NativeWind; same token names, swappable).
  Gotcha: RNTL v14 render() is async - always `await render(...)`.
  Emulator render check deferred to 15.1 (no AVD on this machine yet).
- **15.1 Auth & session**: login/signup/reset screens (same zod schemas),
  Keystore-backed session store, refresh interceptor, AuthGuard routing.
  DoD: full auth loop against the local stack from the emulator.
- **15.2 Workspaces & channels**: workspace switcher, channels list/connect
  (Custom Tabs + deep link)/disconnect with the platform caveats. DoD:
  connect X + IG + TikTok against the simulators from the emulator.
- **15.3 Composer & media**: compose-once editor, per-channel tabs, char
  counter, media-required gate, picker/camera upload with progress, X-style
  preview (FlatList-friendly), link preview card, UTM, shorten-links action.
- **15.4 Scheduling & calendar**: publish-now/slots/specific-time sheet
  (native date-time pickers, tz-correct), month/agenda calendar, cancel.
- **15.5 Analytics**: overview list, per-post breakdown, series chart
  (victory-native), CSV export via share-sheet.
- **15.6 Wallet (read-only) + Integrations + Settings**: balance/ledger/tier
  prices, "top up on the web" hand-off, OGShortener status (key entry stays
  web-only v1), account/appearance/members.
- **15.7 Hardening & release**: Maestro e2e of the core loop against the
  local stack + simulators, accessibility pass (TalkBack, touch targets,
  reduced motion), ProGuard/shrink config, EAS build, Play internal testing.

## 6. Definition of Done (every sub-phase)

- [ ] typecheck + lint + prettier green; unit tests green (hooks AND screens)
- [ ] Verified on a real emulator against the running local stack (curl-level
      claims are not enough; screenshots or Maestro flows recorded)
- [ ] Failure paths exercised (401 refresh, offline, rate limit, validation)
- [ ] Copy free of em dashes; hints present; both themes checked
- [ ] Plan checkboxes updated + memory entry written

## 7. Open questions for the owner

1. Expo/React Native confirmed over native Kotlin? (Recommendation above.)
2. Production deep-link domain (App Links need a hosted
   `assetlinks.json`) - do we own `app.postal.example`-equivalent yet?
3. Wallet v1 read-only acceptable, or fight the Play Billing question now?
4. Google Play developer account: exists, or needs registering ($25 one-time)?
