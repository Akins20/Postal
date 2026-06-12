/**
 * Feature-route frame inside the global chrome: a scroll container with room
 * for the dock. All navigation lives in the header and dock - no side rail.
 */
export default function FeatureLayout({ children }: { children: React.ReactNode }) {
  return (
    <main id="main" className="h-full overflow-auto pb-32">
      {children}
    </main>
  );
}
