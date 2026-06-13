import { useEffect, useState } from "react";

import { refreshSession } from "@/api/client";

/**
 * On cold start, try once to exchange a stored Keystore refresh token for an
 * access token. Returns false until that attempt settles so the app can hold a
 * splash and avoid a login-screen flash for already-signed-in users.
 */
export function useBootstrap(): boolean {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    let active = true;
    refreshSession().finally(() => {
      if (active) setReady(true);
    });
    return () => {
      active = false;
    };
  }, []);
  return ready;
}
