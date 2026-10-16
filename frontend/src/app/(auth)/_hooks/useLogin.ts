import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/app/lib/api";
import { ApiReturn } from "@/app/types/api";
import { useAuth } from "@/context/AuthContext";
import Cookies from "universal-cookie";

const cookies = new Cookies();

// Types for login
export type LoginRequest = {
  username: string;
  password: string;
};

export type LoginResponse = {
  id: number;
  username: string;
  token: string;
};

export type LoginError = {
  response?: {
    data?: {
      error?: string;
    };
  };
  message?: string;
};

// Hook for login functionality
export function useLogin() {
  const { setUser } = useAuth();

  return useMutation<LoginResponse, LoginError, LoginRequest>({
    mutationFn: async (credentials: LoginRequest) => {
      const response = await api.post<ApiReturn<LoginResponse>>(
        "/auth/login",
        {
          username: credentials.username,
          password: credentials.password,
        },
        {
          headers: {
            "Content-Type": "application/json",
          },
        }
      );

      if (!response.data) {
        throw new Error("Failed to login");
      }

      return response.data.data;
    },
    onSuccess: (userData) => {
      // Store token in cookie (manually from frontend)
      cookies.set("Authorization", userData.token, {
        path: "/",
        maxAge: 3600 * 6, // 6 hours
        sameSite: "lax",
      });

      // Set user data in auth context (without token)
      setUser({
        id: userData.id,
        username: userData.username,
      });

      // Show success message
      toast.success("Login berhasil!");

      // Navigate to dashboard with window.location to ensure full page reload
      // This prevents RSC issues and ensures middleware runs properly
      const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";
      window.location.href = `${basePath}/dashboard`;
    },
    onError: (error) => {
      // Handle login error
      const errorMessage = error.response?.data?.error || error.message || "Login gagal";
      toast.error(errorMessage);
    },
  });
}

// Hook for logout functionality
export function useLogout() {
  const { setUser } = useAuth();

  return useMutation<void, LoginError, void>({
    mutationFn: async () => {
      const response = await api.post("/auth/logout");

      if (!response.data) {
        throw new Error("Failed to logout");
      }
    },
    onSuccess: () => {
      // Remove token cookie
      cookies.remove("Authorization", { path: "/" });

      // Clear user data
      setUser(null);

      // Show success message
      toast.success("Logout berhasil!");

      // Navigate to login with window.location to ensure full page reload
      const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";
      window.location.href = `${basePath}/login`;
    },
    onError: (error) => {
      // Handle logout error
      const errorMessage = error.response?.data?.error || error.message || "Logout gagal";
      toast.error(errorMessage);

      // Even if logout fails on server, clear local state
      cookies.remove("Authorization", { path: "/" });
      setUser(null);
      const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";
      window.location.href = `${basePath}/login`;
    },
  });
}
