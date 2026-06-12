import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import { useState } from "react";
import { StatusBar } from "expo-status-bar";

import { usePalette } from "@/lib/use-palette";

/**
 * Root layout: query client, themed status bar, and the navigation stack.
 * Auth gating (redirect to login) arrives in 15.1.
 */
export default function RootLayout() {
  // One client per app instance; avoids re-creation on re-render.
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
      }),
  );
  const { palette, scheme } = usePalette();

  return (
    <QueryClientProvider client={queryClient}>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      <Stack
        screenOptions={{
          headerShown: false,
          contentStyle: { backgroundColor: palette.surface },
        }}
      >
        <Stack.Screen name="(tabs)" />
      </Stack>
    </QueryClientProvider>
  );
}
