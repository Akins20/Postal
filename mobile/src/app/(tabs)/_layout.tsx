import { Redirect, Tabs } from "expo-router";
import { BarChart3, Calendar, Home, Radio, SquarePen } from "lucide-react-native";

import { useMe } from "@/data/auth";
import { usePalette } from "@/lib/use-palette";
import { BrandSplash } from "@/ui/brand-splash";

/**
 * The dock's mobile counterpart: a five-item bottom tab bar (Home, Compose,
 * Calendar, Channels, More). Media/Analytics/Wallet/Integrations/Settings
 * live behind More - nine items don't fit a phone tab bar honestly.
 */
export default function TabLayout() {
  const { palette } = usePalette();
  const { data: user, isPending } = useMe();

  // Auth guard: bounce signed-out users to login.
  if (isPending) return <BrandSplash />;
  if (!user) return <Redirect href="/login" />;

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: palette.accent,
        tabBarInactiveTintColor: palette.fgSubtle,
        tabBarStyle: {
          backgroundColor: palette.vibrancyDock,
          borderTopColor: palette.separator,
        },
        sceneStyle: { backgroundColor: palette.surface },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: "Home",
          tabBarIcon: ({ color, size }) => <Home color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="compose"
        options={{
          title: "Compose",
          tabBarIcon: ({ color, size }) => <SquarePen color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="calendar"
        options={{
          title: "Calendar",
          tabBarIcon: ({ color, size }) => <Calendar color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="channels"
        options={{
          title: "Channels",
          tabBarIcon: ({ color, size }) => <Radio color={color} size={size} />,
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: "More",
          tabBarIcon: ({ color, size }) => <BarChart3 color={color} size={size} />,
        }}
      />
    </Tabs>
  );
}
