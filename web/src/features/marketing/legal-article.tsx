import type { ReactNode } from "react";

/**
 * Shared layout for legal/marketing long-form pages. Renders a centered reading
 * column and styles plain child elements (h2, p, ul, strong, a) so each page can
 * stay simple markup. No prose plugin dependency.
 */
export function LegalArticle({
  title,
  updated,
  intro,
  children,
}: {
  title: string;
  updated?: string;
  intro?: ReactNode;
  children: ReactNode;
}) {
  return (
    <article className="mx-auto max-w-3xl px-6 py-14 sm:py-20">
      <h1 className="text-fg text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h1>
      {updated && <p className="text-fg-subtle mt-3 text-sm">Last updated: {updated}</p>}
      {intro && <p className="text-fg-muted mt-5 text-base leading-relaxed">{intro}</p>}
      <div className="text-fg-muted [&_a]:text-accent [&_h2]:text-fg [&_strong]:text-fg mt-8 flex flex-col gap-4 text-sm leading-relaxed [&_a]:hover:underline [&_h2]:mt-8 [&_h2]:text-lg [&_h2]:font-semibold [&_li]:ml-1 [&_ul]:flex [&_ul]:list-disc [&_ul]:flex-col [&_ul]:gap-2 [&_ul]:pl-5">
        {children}
      </div>
    </article>
  );
}
