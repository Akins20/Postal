import { LegalArticle } from "@/features/marketing/legal-article";

export const metadata = {
  title: "Terms of Service",
  description: "The terms that govern your use of Postal.",
};

export default function TermsPage() {
  return (
    <LegalArticle
      title="Terms of Service"
      updated="June 13, 2026"
      intro="These terms govern your use of Postal. By creating an account or using the service, you agree to them."
    >
      <h2>1. The service</h2>
      <p>
        Postal lets you connect social media accounts and schedule and publish posts to them. The
        service is provided as is. We work to keep it reliable, but we do not guarantee that it will
        be uninterrupted or error free.
      </p>

      <h2>2. Your account</h2>
      <p>
        You are responsible for keeping your credentials secure and for all activity under your
        account. You must provide a valid email address and verify it. You must be old enough to use
        the connected platforms under their own rules.
      </p>

      <h2>3. Your content and connected accounts</h2>
      <p>
        You keep ownership of everything you publish through Postal. You grant us the limited rights
        needed to store your content and deliver it to the platforms you choose. You are responsible
        for your content and for complying with the terms and policies of each platform you connect,
        including X, Instagram, and TikTok.
      </p>

      <h2>4. Wallet and X publishing</h2>
      <p>
        Postal is free to use. Publishing to X is billed by X per request, so Postal charges those
        publishes against a prepaid wallet. Credits are priced by the type of post. Wallet top ups
        are non refundable except where required by law, and unused credits remain available in your
        workspace.
      </p>

      <h2>5. Acceptable use</h2>
      <ul>
        <li>Do not use Postal to send spam or to violate any platform&apos;s rules.</li>
        <li>Do not publish unlawful, infringing, or harmful content.</li>
        <li>Do not attempt to disrupt, reverse engineer, or abuse the service.</li>
      </ul>
      <p>We may suspend or close accounts that break these rules.</p>

      <h2>6. Termination</h2>
      <p>
        You can stop using Postal and delete your account at any time. We may suspend or end access
        if you breach these terms or if we must do so to protect the service or comply with the law.
      </p>

      <h2>7. Liability</h2>
      <p>
        To the extent permitted by law, Postal is not liable for indirect or consequential losses,
        or for content delivered to or removed by third party platforms. Our total liability is
        limited to the amount you paid into your wallet in the prior twelve months.
      </p>

      <h2>8. Changes</h2>
      <p>
        We may update these terms. If we make a material change, we will notify you. Continued use
        after a change means you accept the updated terms.
      </p>
    </LegalArticle>
  );
}
