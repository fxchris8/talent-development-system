import { BASE_URL } from "./api";

// True only for a usable backend-relative path such as "/storage/assessments/x.png"
export const isValidImageUrl = (imageUrl: string | null | undefined): boolean => {
  if (!imageUrl || typeof imageUrl !== "string") return false;
  const trimmed = imageUrl.trim();
  if (trimmed === "" || trimmed === "null" || trimmed === "undefined") return false;
  return trimmed.startsWith("/");
};

// Build the absolute URL of an image served by the backend (uploads live under /storage).
export const resolveImageUrl = (imageUrl: string): string =>
  `${BASE_URL.replace(/\/+$/, "")}${imageUrl.trim()}`;
