import { NextResponse } from "next/server";
import { NextRequest } from "next/server";
import { jwtVerify } from "jose";

const TOKEN_COOKIE = "Authorization";
const secret = new TextEncoder().encode(process.env.JWT_SECRET?.trim());

// A token counts only when it is signed with our secret and not past its "expired" claim
// (the backend uses that claim instead of the standard "exp").
async function hasValidSession(token: string | undefined): Promise<boolean> {
  if (!token) return false;
  try {
    const { payload } = await jwtVerify(token, secret, { algorithms: ["HS256"] });
    const expired = payload.expired;
    return typeof expired === "number" && Date.now() / 1000 <= expired;
  } catch {
    return false;
  }
}

// Send the visitor to the login page and drop a token that no longer works. Without the
// delete, /login would see "a token" and bounce back to /dashboard forever.
function redirectToLogin(request: NextRequest, hadToken: boolean) {
  const response = NextResponse.redirect(new URL("/login", request.url));
  if (hadToken) response.cookies.delete(TOKEN_COOKIE);
  return response;
}

export async function proxy(request: NextRequest) {
  const token = request.cookies.get(TOKEN_COOKIE)?.value;
  const url = request.nextUrl;
  const loggedIn = await hasValidSession(token);

  if (url.pathname === "/") {
    return loggedIn
      ? NextResponse.redirect(new URL("/dashboard", request.url))
      : redirectToLogin(request, Boolean(token));
  }

  if (url.pathname.startsWith("/login")) {
    if (loggedIn) {
      return NextResponse.redirect(new URL("/dashboard", request.url));
    }
    // Show the login form; a stale token is removed so it cannot cause trouble later.
    const response = NextResponse.next();
    if (token) response.cookies.delete(TOKEN_COOKIE);
    return response;
  }

  const protectedRoutes = [
    "/dashboard",
    "/master-report",
    "/assessment-manager",
    "/assessment-type-manager",
    "/assignments",
    "/report-mentoring",
    "/new-recruiter-report",
    "/quiz-history",
    "/batch-control",
    "/admin-panel",
  ];

  const isProtectedRoute = protectedRoutes.some((route) => url.pathname.startsWith(route));

  if (isProtectedRoute && !loggedIn) {
    return redirectToLogin(request, Boolean(token));
  }
  return NextResponse.next();
}

export const config = {
  matcher: [
    "/",
    "/login",
    "/dashboard/:path*",
    "/master-report/:path*",
    "/assessment-manager/:path*",
    "/assessment-type-manager/:path*",
    "/crew-evaluation-system/:path*",
    "/assignments/:path*",
    "/value-assessment/:path*",
    "/report-mentoring/:path*",
    "/new-recruiter-report/:path*",
    "/quiz-history/:path*",
    "/batch-control/:path*",
    "/admin-panel/:path*",
  ],
};
