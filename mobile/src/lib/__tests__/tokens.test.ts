import { palettes } from "@/lib/tokens";

describe("design tokens", () => {
  it("defines the same semantic names for both themes", () => {
    expect(Object.keys(palettes.light).sort()).toEqual(Object.keys(palettes.dark).sort());
  });

  it("keeps light and dark visually distinct", () => {
    expect(palettes.light.surface).not.toBe(palettes.dark.surface);
    expect(palettes.light.fg).not.toBe(palettes.dark.fg);
  });
});
