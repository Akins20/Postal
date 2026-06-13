import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import { SafeAreaProvider } from "react-native-safe-area-context";

import { useBootstrap } from "@/features/auth/use-bootstrap";
import { usePalette } from "@/lib/use-palette";
import { BrandSplash } from "@/ui/brand-splash";

/**
 * Root layout: query client + safe-area, then a session-bootstrap gate (try a
 * Keystore refresh before first paint) wrapping the two route groups -
 * (auth) public, (tabs) signed-in. Each group's layout handles its own
 * redirect.
 */
export default function RootLayout() {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: 1, staleTime: 30_000 } } }),
  );
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <SessionGate />
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

function SessionGate() {
  const ready = useBootstrap();
  const { palette, scheme } = usePalette();
  if (!ready) return <BrandSplash />;
  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      <Stack
        screenOptions={{ headerShown: false, contentStyle: { backgroundColor: palette.surface } }}
      >
        <Stack.Screen name="(auth)" />
        <Stack.Screen name="(tabs)" />
      </Stack>
    </>
  );
}
