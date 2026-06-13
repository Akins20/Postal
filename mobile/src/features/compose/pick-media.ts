import * as ImagePicker from "expo-image-picker";

import type { PickedFile } from "@/data/media";

/**
 * Launch the system image/video picker and return a file ready to upload, or
 * null if the user cancelled. Requests library permission on first use.
 */
export async function pickMedia(): Promise<PickedFile | null> {
  const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!perm.granted) return null;
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images", "videos"],
    quality: 0.9,
  });
  if (result.canceled || result.assets.length === 0) return null;
  const a = result.assets[0];
  const name = a.fileName ?? a.uri.split("/").pop() ?? "upload";
  const mime = a.mimeType ?? (a.type === "video" ? "video/mp4" : "image/jpeg");
  return { uri: a.uri, name, mime };
}
