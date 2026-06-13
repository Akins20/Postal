import { cn } from "@/lib/cn";

/**
 * The Postal paper-plane brand mark (the same geometry as the mobile logo and
 * the app-router icons). `tone` picks the rendering: "accent" draws the plane in
 * the brand blue on a transparent ground (for light surfaces); "onAccent" draws
 * it white (for use on a brand-blue background).
 */
export function Logo({
  className,
  tone = "accent",
}: {
  className?: string;
  tone?: "accent" | "onAccent";
}) {
  const faceA = tone === "onAccent" ? "#ffffff" : "var(--accent)";
  const faceB = tone === "onAccent" ? "rgba(255,255,255,0.78)" : "var(--accent-soft)";
  return (
    <svg
      viewBox="0 0 100 100"
      className={cn("size-8", className)}
      role="img"
      aria-label="Postal"
      fill="none"
    >
      <path d="M93 8 L7 46 L40 60 Z" fill={faceA} />
      <path d="M93 8 L40 60 L52 93 Z" fill={faceB} />
    </svg>
  );
}
