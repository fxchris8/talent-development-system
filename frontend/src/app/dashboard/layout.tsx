import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import BreadcrumbClient from "./BreadcrumbClient"; // ✅ Tambahkan ini
import React from "react";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <main className="my-3 mx-4 mt-10 min-w-0 flex-1">
        <BreadcrumbClient /> {/* ✅ Breadcrumb dipecah jadi client */}
        {children}
      </main>
    </SidebarProvider>
  );
}
