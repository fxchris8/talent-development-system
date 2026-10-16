"use client";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import React, { Suspense } from "react";
import { useSearchParams } from "next/navigation";

// Form validation GOES HERE
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useLogin } from "../_hooks/useLogin";
const FormSchema = z.object({
  username: z.string().min(1, {
    message: "Username is required",
  }),
  password: z.string().min(1, {
    message: "Password is required",
  }),
});
// Form validation END HERE

function PageContent() {
  const loginMutation = useLogin();
  const searchParams = useSearchParams();

  const form = useForm<z.infer<typeof FormSchema>>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      username: "",
      password: "",
    },
  });

  const onSubmit = (data: z.infer<typeof FormSchema>) => {
    // Use React Query mutation for login
    loginMutation.mutate({
      username: data.username,
      password: data.password,
    });
  };

  const handleSsoLogin = React.useCallback(() => {
    const apiBase = process.env.NEXT_PUBLIC_API_ENDPOINT || window.location.origin;
    const params = new URLSearchParams();
    const clientID = searchParams.get("client_id") || process.env.NEXT_PUBLIC_SSO_CLIENT_ID;
    if (clientID) {
      params.set("client_id", clientID);
    }

    const queryString = params.toString();
    const target = `${apiBase}/api/auth/sso/initiate${queryString ? `?${queryString}` : ""}`;
    window.location.href = target;
  }, [searchParams]);

  React.useEffect(() => {
    const shouldAutoSSO = searchParams.get("login_sso") === "true";
    const clientID = searchParams.get("client_id");
    const configuredID = process.env.NEXT_PUBLIC_SSO_CLIENT_ID;
    if (shouldAutoSSO) {
      if (!configuredID || clientID === configuredID || !clientID) {
        handleSsoLogin();
      }
    }
  }, [handleSsoLogin, searchParams]);

  const ssoError = searchParams.get("sso_error");

  return (
    <div className="flex flex-col gap-5">
      <h1 className="font-bold text-3xl">Talent Development System</h1>
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Login to your account</CardTitle>
          <CardDescription>
            Enter your username and password below to login to your account
          </CardDescription>
        </CardHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <CardContent>
              <div className="flex flex-col gap-4">
                {/* USERNAME */}
                <div className="grid gap-2">
                  <FormField
                    control={form.control}
                    name="username"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Username</FormLabel>
                        <FormControl>
                          <Input placeholder="Your username" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                {/* PASSWORD */}
                <div className="grid gap-1">
                  <FormField
                    control={form.control}
                    name="password"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Password</FormLabel>
                        <FormControl>
                          <Input placeholder="******" type="password" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            </CardContent>
            <CardFooter className="flex-col gap-2 mt-4">
              {ssoError ? (
                <div className="w-full rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  SSO login failed: {ssoError}
                </div>
              ) : null}
              <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
                {loginMutation.isPending ? "Loading..." : "Login"}
              </Button>
              <Button type="button" variant="outline" className="w-full" onClick={handleSsoLogin}>
                Login with SSO SPIL
              </Button>
            </CardFooter>
          </form>
        </Form>
      </Card>
    </div>
  );
}

// useSearchParams() membuat halaman ini bergantung pada URL saat render di client. Tanpa
// Suspense, `next build` (production) gagal mem-prerender halaman ini dan build berhenti.
export default function Page() {
  return (
    <Suspense fallback={null}>
      <PageContent />
    </Suspense>
  );
}
