import * as SecureStore from "expo-secure-store";

/**
 * Session storage. The access token (short-lived) lives in MEMORY only and
 * never touches disk; the refresh token (long-lived) lives in the Android
 * Keystore via expo-secure-store. This is the mobile counterpart of the web's
 * httpOnly cookies - the browser kept tokens out of JS, here the Keystore
 * keeps the refresh token out of plain storage. No network here (avoids an
 * import cycle with the API client).
 */

const REFRESH_KEY = "postal.refreshToken";

let accessToken: string | null = null;

/** The in-memory access token (null when signed out). */
export function getAccessToken(): string | null {
  return accessToken;
}

/** Set the in-memory access token (null clears it). */
export function setAccessToken(token: string | null): void {
  accessToken = token;
}

/** Read the persisted refresh token from the Keystore. */
export async function getRefreshToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(REFRESH_KEY);
  } catch {
    return null;
  }
}

/** Persist (or, with null, delete) the refresh token in the Keystore. */
export async function setRefreshToken(token: string | null): Promise<void> {
  try {
    if (token) {
      await SecureStore.setItemAsync(REFRESH_KEY, token);
    } else {
      await SecureStore.deleteItemAsync(REFRESH_KEY);
    }
  } catch {
    // Keystore unavailable (e.g. tests): in-memory access token still works.
  }
}

/** Store a freshly issued token pair (access in memory, refresh in Keystore). */
export async function saveSession(access: string, refresh: string | undefined): Promise<void> {
  setAccessToken(access);
  if (refresh) await setRefreshToken(refresh);
}

/** Clear both tokens (logout / unrecoverable refresh failure). */
export async function clearSession(): Promise<void> {
  setAccessToken(null);
  await setRefreshToken(null);
}
