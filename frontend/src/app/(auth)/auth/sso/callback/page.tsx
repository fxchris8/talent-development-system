"use client";

import { useEffect, Suspense } from "react";
import { useSearchParams } from "next/navigation";
import Cookies from "universal-cookie";
import { Loader2, ShieldCheck } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const cookies = new Cookies();

type TokenPayload = {
  sub?: number | string;
  username?: string;
};

function decodeTokenPayload(token: string): TokenPayload | null {
  try {
    const parts = token.split(".");
    if (parts.length < 2) {
      return null;
    }

    const payload = parts[1];
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const decoded = atob(normalized);
    return JSON.parse(decoded) as TokenPayload;
  } catch {
    return null;
  }
}

function SSOCallbackPageContent() {
  const searchParams = useSearchParams();

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) {
      window.location.href = "/login?sso_error=Missing+token+from+callback";
      return;
    }

    cookies.set("Authorization", token, {
      path: "/",
      maxAge: 3600 * 6,
      sameSite: "lax",
    });

    const payload = decodeTokenPayload(token);
    if (payload?.sub !== undefined && payload?.username) {
      localStorage.setItem(
        "USER",
        JSON.stringify({
          id: Number(payload.sub),
          username: payload.username,
        })
      );
    }

    window.location.href = "/dashboard";
  }, [searchParams]);

  return (
    <div className="relative min-h-screen w-full overflow-hidden bg-slate-50">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(14,165,233,0.08),transparent_30%),radial-gradient(circle_at_80%_0%,rgba(16,185,129,0.08),transparent_35%)]" />

      <div className="relative flex min-h-screen items-center justify-center px-4">
        <Card className="w-full max-w-md border-slate-200 shadow-lg">
          <CardHeader className="space-y-3">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
              <ShieldCheck className="h-6 w-6" />
            </div>
            <CardTitle className="text-center text-xl">Signing in with SSO</CardTitle>
            <CardDescription className="text-center">
              Verifying your token and preparing your session.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-center gap-2 rounded-md border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600">
              <Loader2 className="h-4 w-4 animate-spin" />
              Please wait a moment...
            </div>
            <p className="text-center text-xs text-slate-500">
              You will be redirected to the dashboard automatically.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// useSearchParams() membuat halaman ini bergantung pada URL saat render di client. Tanpa
// Suspense, `next build` (production) gagal mem-prerender halaman ini dan build berhenti.
export default function SSOCallbackPage() {
  return (
    <Suspense fallback={null}>
      <SSOCallbackPageContent />
    </Suspense>
  );
}
