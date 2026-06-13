import { Redirect, Stack } from "expo-router";

import { useMe } from "@/data/auth";
import { BrandSplash } from "@/ui/brand-splash";

/** Public auth routes: signed-in users are bounced to the app. */
export default function AuthLayout() {
  const { data: user, isPending } = useMe();
  if (isPending) return <BrandSplash />;
  if (user) return <Redirect href="/" />;
  return <Stack screenOptions={{ headerShown: false }} />;
}
