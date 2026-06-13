import { Stack } from "expo-router";

import { usePalette } from "@/lib/use-palette";

/** The "More" section: a stack hosting Analytics, Wallet, and Settings. */
export default function MoreLayout() {
  const { palette } = usePalette();
  return (
    <Stack
      screenOptions={{
        headerStyle: { backgroundColor: palette.surface },
        headerTintColor: palette.fg,
        headerShadowVisible: false,
        contentStyle: { backgroundColor: palette.surface },
      }}
    >
      <Stack.Screen name="index" options={{ headerShown: false }} />
      <Stack.Screen name="analytics" options={{ title: "Analytics" }} />
      <Stack.Screen name="wallet" options={{ title: "Wallet" }} />
      <Stack.Screen name="settings" options={{ title: "Settings" }} />
    </Stack>
  );
}
