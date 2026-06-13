import Link from "next/link";

import { LegalArticle } from "@/features/marketing/legal-article";

export const metadata = {
  title: "About",
  description:
    "Postal is a free, no-paywall social media scheduling and publishing platform for X, Instagram, and TikTok.",
};

export default function AboutPage() {
  return (
    <LegalArticle
      title="About Postal"
      intro="Postal is a free, no-paywall tool for scheduling and publishing to your social channels. Compose a post once, tailor it per platform, and let Postal publish it on your schedule."
    >
      <h2>What you can do</h2>
      <ul>
        <li>Connect X, Instagram, and TikTok and manage them from one calendar.</li>
        <li>Write a post once and adapt the wording, media, and links per channel.</li>
        <li>Schedule into open time slots, pick an exact time, or publish right now.</li>
        <li>See link previews and media exactly as they will appear once published.</li>
        <li>Track how posts perform with per-channel analytics.</li>
      </ul>

      <h2>Why it is free</h2>
      <p>
        The core product has no paywall. The only paid action is publishing to X, which charges per
        request through its API. Postal passes that cost through with a simple prepaid wallet,
        priced by the kind of post, so you only pay for what X actually bills. Everything else,
        across every other platform, is free.
      </p>

      <h2>Built to respect your accounts</h2>
      <p>
        Postal connects to each platform through its official authorization flow. Your access tokens
        are encrypted at rest, and you can disconnect a channel at any time. We never post without
        your instruction.
      </p>

      <p>
        Ready to start? <Link href="/signup">Create a free account</Link> or{" "}
        <Link href="/login">sign in</Link>.
      </p>
    </LegalArticle>
  );
}
