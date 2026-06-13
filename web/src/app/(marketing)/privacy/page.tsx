import { LegalArticle } from "@/features/marketing/legal-article";

export const metadata = {
  title: "Privacy Policy",
  description: "How Postal collects, uses, and protects your data.",
};

export default function PrivacyPage() {
  return (
    <LegalArticle
      title="Privacy Policy"
      updated="June 13, 2026"
      intro="This policy explains what data Postal collects, how we use it, and the choices you have."
    >
      <h2>Data we collect</h2>
      <ul>
        <li>
          <strong>Account data:</strong> your email address and a securely hashed password.
        </li>
        <li>
          <strong>Connected accounts:</strong> access tokens for the social platforms you connect,
          stored encrypted at rest.
        </li>
        <li>
          <strong>Content:</strong> the posts, media, and schedules you create.
        </li>
        <li>
          <strong>Usage and analytics:</strong> post performance metrics fetched from the platforms
          and basic logs needed to operate the service.
        </li>
      </ul>

      <h2>How we use it</h2>
      <p>
        We use your data to operate Postal: to authenticate you, to publish on your behalf, to show
        your analytics, and to keep the service secure. We do not sell your data, and we do not use
        your content to train models.
      </p>

      <h2>Service providers</h2>
      <p>We share data only with the providers needed to run the service:</p>
      <ul>
        <li>
          <strong>Resend</strong> to send account verification and password reset email.
        </li>
        <li>
          <strong>Stripe</strong> and <strong>Paystack</strong> to process wallet top ups. Postal
          does not store your card details.
        </li>
        <li>
          <strong>Cloudflare R2</strong> to store media you upload.
        </li>
        <li>
          The <strong>social platforms</strong> you connect (X, Instagram, TikTok), to publish and
          read analytics at your instruction.
        </li>
      </ul>

      <h2>Token security</h2>
      <p>
        Access tokens for your connected accounts are protected with envelope encryption. They are
        used only to perform actions you ask for, and they are removed when you disconnect a
        channel.
      </p>

      <h2>Retention</h2>
      <p>
        We keep your data while your account is active. Analytics snapshots are retained for a
        limited window. When you delete your account, we delete your personal data and revoke stored
        tokens, except where we must keep records to comply with the law.
      </p>

      <h2>Your choices</h2>
      <ul>
        <li>Disconnect any channel at any time to revoke its stored token.</li>
        <li>Delete your account to remove your personal data.</li>
        <li>Contact us to request a copy of your data or to ask a privacy question.</li>
      </ul>

      <h2>Contact</h2>
      <p>
        Privacy questions can be sent to{" "}
        <a href="mailto:support@lettstv.com">support@lettstv.com</a>.
      </p>
    </LegalArticle>
  );
}
