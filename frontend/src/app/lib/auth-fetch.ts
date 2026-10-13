import Cookies from "universal-cookie";

// fetch() that sends the login token, for endpoints that require authentication.
export function authFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const token = new Cookies().get("Authorization");
  const headers = new Headers(init.headers);
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return fetch(input, { ...init, headers, credentials: "include" });
}
