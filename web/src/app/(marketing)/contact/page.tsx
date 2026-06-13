import { LegalArticle } from "@/features/marketing/legal-article";

export const metadata = {
  title: "Contact",
  description: "Get in touch with the Postal team.",
};

export default function ContactPage() {
  return (
    <LegalArticle
      title="Contact"
      intro="We are happy to help with questions, feedback, or account issues."
    >
      <h2>Support</h2>
      <p>
        Email <a href="mailto:support@lettstv.com">support@lettstv.com</a> and we will get back to
        you. Including the email on your account and a clear description of the issue helps us
        respond faster.
      </p>

      <h2>Privacy and legal</h2>
      <p>
        For privacy requests, see the <a href="/privacy">Privacy Policy</a>. For terms questions,
        see the <a href="/terms">Terms of Service</a>.
      </p>
    </LegalArticle>
  );
}
