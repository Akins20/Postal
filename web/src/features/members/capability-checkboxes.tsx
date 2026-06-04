"use client";

import { CAPABILITIES, type Capability } from "@/config/capabilities";

/** A controlled checkbox group for picking custom capabilities (FRONTEND_PLAN §12.2). */
export function CapabilityCheckboxes({
  value,
  onChange,
}: {
  value: Capability[];
  onChange: (next: Capability[]) => void;
}) {
  const selected = new Set(value);
  const toggle = (cap: Capability) => {
    const next = new Set(selected);
    if (next.has(cap)) next.delete(cap);
    else next.add(cap);
    onChange([...next]);
  };

  return (
    <fieldset className="flex flex-col gap-1">
      <legend className="sr-only">Capabilities</legend>
      {CAPABILITIES.map((c) => (
        <label key={c.value} className="flex items-start gap-2.5 rounded-md p-1 text-sm">
          <input
            type="checkbox"
            checked={selected.has(c.value)}
            onChange={() => toggle(c.value)}
            className="border-separator text-accent focus-visible:ring-ring mt-0.5 h-4 w-4 rounded focus-visible:ring-2"
          />
          <span className="flex flex-col">
            <span className="text-fg font-medium">{c.label}</span>
            <span className="text-fg-muted text-xs">{c.description}</span>
          </span>
        </label>
      ))}
    </fieldset>
  );
}
