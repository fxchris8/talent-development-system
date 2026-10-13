"use client";

import { useEffect, useState } from "react";
import {
  LaptopMinimal,
  ChevronUp,
  Inbox,
  User2,
  Settings,
  Calendar,
  Users,
  History,
} from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Badge } from "./ui/badge";
import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/context/AuthContext";
import { Button } from "./ui/button";
import { useLogout } from "@/app/(auth)/_hooks/useLogin";
import { useGetOverdueCount } from "@/app/dashboard/training-plan/_hooks/useTrainingPlan";

// Menu items.
const items = [
  {
    title: "Dashboard",
    url: "/dashboard",
    icon: LaptopMinimal,
  },
  {
    title: "Master Report",
    url: "/master-report",
    icon: LaptopMinimal,
  },
  {
    title: "New Recruiter Report",
    url: "/new-recruiter-report",
    icon: Users,
  },
  {
    title: "Upload Excel",
    url: "/dashboard/excel",
    icon: Inbox,
  },
  {
    title: "Training Plan",
    url: "/dashboard/training-plan",
    icon: Calendar,
  },
  {
    title: "Assessment Manager",
    url: "/assessment-manager",
    icon: Settings,
  },
  {
    title: "Assessment Activation",
    url: "/assessment-type-manager",
    icon: Settings,
  },
  {
    title: "User Management",
    url: "/dashboard/users",
    icon: Users,
  },
  {
    title: "Quiz History",
    url: "/quiz-history",
    icon: History,
  },
  {
    title: "Batch Control",
    url: "/batch-control",
    icon: Calendar,
  },
];

export function AppSidebar() {
  const [mounted, setMounted] = useState(false);
  const pathname = usePathname();
  const { user } = useAuth();
  const logoutMutation = useLogout();

  // Fetch overdue count for all programs
  const { data: sdpCount } = useGetOverdueCount("SDP");
  const { data: mdpCount } = useGetOverdueCount("MDP");
  const { data: fdpCount } = useGetOverdueCount("FDP");

  // Calculate total overdue across all programs
  const totalOverdueCount = (sdpCount || 0) + (mdpCount || 0) + (fdpCount || 0);

  useEffect(() => {
    setMounted(true);
  }, []);

  return (
    <Sidebar collapsible="icon" variant="floating">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem className="py-2">
            <SidebarMenuButton className="py-6 px-2">
              <div className="flex gap-x-2 items-center">
                <Image
                  src="/images/spil_logo.svg"
                  alt="Logo SPIL"
                  width={34}
                  height={34}
                  className="mr-2 h-[34px] w-[34px] shrink-0 object-contain"
                />
                <h2 className="text-2xl font-bold text-center">SPIL</h2>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Menu</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild isActive={pathname === item.url}>
                    <Link key={item.url} href={item.url}>
                      <item.icon />
                      <span className="flex-1">{item.title}</span>
                      {item.title === "Training Plan" && totalOverdueCount > 0 && (
                        <Badge
                          variant="destructive"
                          className="ml-auto h-4 w-4 rounded-full p-0 flex items-center justify-center text-[10px] font-medium"
                        >
                          {totalOverdueCount}
                        </Badge>
                      )}
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            {mounted ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton>
                    <User2 /> {user?.username}
                    <ChevronUp className="ml-auto" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent side="top" className="w-[--radix-popper-anchor-width]">
                  <DropdownMenuItem className="p-0">
                    <Button
                      className="w-full"
                      variant="destructive"
                      disabled={logoutMutation.isPending}
                      onClick={() => logoutMutation.mutate()}
                    >
                      {logoutMutation.isPending ? "Signing out..." : "Sign out"}
                    </Button>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <SidebarMenuButton>
                <User2 /> {user?.username}
                <ChevronUp className="ml-auto" />
              </SidebarMenuButton>
            )}
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
