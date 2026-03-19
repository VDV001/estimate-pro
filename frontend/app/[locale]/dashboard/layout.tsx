"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { usePathname, useRouter } from "@/i18n/navigation";
import { AuthGuard } from "@/features/auth/components/auth-guard";
import { useAuthStore } from "@/features/auth/store";
import {
  LayoutDashboard,
  FolderKanban,
  Settings,
  LogOut,
  Bell,
} from "lucide-react";
import {
  Sidebar,
  SidebarBody,
  SidebarLink,
} from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { LocaleToggle } from "@/components/ui/locale-toggle";
import { UserAvatar } from "@/components/ui/user-avatar";
import { HoverCard, HoverCardTrigger, HoverCardContent } from "@/components/ui/hover-card";
import { Toaster } from "@/components/ui/sonner";
import { useWebSocket } from "@/hooks/use-websocket";
import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const t = useTranslations();
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const user = useAuthStore((s) => s.user);
  const logoutUser = useAuthStore((s) => s.logoutUser);

  const handleLogout = () => {
    logoutUser();
    router.push("/");
  };

  const isActive = (href: string) =>
    href === "/dashboard"
      ? pathname === "/dashboard" || pathname === "/en/dashboard"
      : pathname?.startsWith(href);

  const links = [
    {
      label: t("dashboard.overview"),
      href: "/dashboard",
      icon: <LayoutDashboard className="h-5 w-5 flex-shrink-0" />,
      active: isActive("/dashboard"),
    },
    {
      label: t("dashboard.projects"),
      href: "/dashboard/projects",
      icon: <FolderKanban className="h-5 w-5 flex-shrink-0" />,
      active: isActive("/dashboard/projects"),
    },
    {
      label: t("dashboard.notifications"),
      href: "/dashboard/notifications",
      icon: <Bell className="h-5 w-5 flex-shrink-0" />,
      active: isActive("/dashboard/notifications"),
    },
    {
      label: t("dashboard.settings"),
      href: "/dashboard/settings",
      icon: <Settings className="h-5 w-5 flex-shrink-0" />,
      active: isActive("/dashboard/settings"),
    },
  ];

  useWebSocket();

  return (
    <AuthGuard>
      <Toaster />
      <div className="flex flex-col h-screen overflow-hidden bg-background">
      {/* Top header bar — full width */}
      <header className="flex items-center justify-between py-3 pr-4 bg-background z-10 flex-shrink-0">
        <div className="flex items-center">
          <div className="w-[52px] flex-shrink-0 flex items-center justify-center">
            <div className="h-8 w-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-xs font-bold text-primary-foreground">
                EP
              </span>
            </div>
          </div>
          <span className="text-lg font-medium text-foreground">
            EstimatePro
          </span>
        </div>
        <div className="flex items-center gap-3">
          <LocaleToggle />
          <ThemeToggle />
          <HoverCard openDelay={200} closeDelay={100}>
            <HoverCardTrigger asChild>
              <button className="rounded-full focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2">
                <UserAvatar name={user?.name} avatarUrl={user?.avatar_url} size="sm" />
              </button>
            </HoverCardTrigger>
            <HoverCardContent align="end" className="w-72 p-5">
              <div className="flex items-center gap-4">
                <UserAvatar name={user?.name} avatarUrl={user?.avatar_url} size="md" />
                <div className="space-y-0.5 min-w-0">
                  <p className="text-sm font-semibold truncate">{user?.name}</p>
                  <p className="text-xs text-muted-foreground truncate">{user?.email}</p>
                </div>
              </div>
              <div className="flex gap-2 mt-6 pt-4 border-t border-border">
                <Link href="/dashboard/settings" className="flex-1">
                  <Button variant="outline" size="sm" className="w-full gap-1.5">
                    <Settings className="h-3.5 w-3.5" />
                    {t("dashboard.settings")}
                  </Button>
                </Link>
                <Button variant="outline" size="sm" className="gap-1.5" onClick={handleLogout}>
                  <LogOut className="h-3.5 w-3.5" />
                </Button>
              </div>
            </HoverCardContent>
          </HoverCard>
        </div>
      </header>

      {/* Sidebar + content below header */}
      <div className="flex flex-1 overflow-hidden">
        <Sidebar open={open} setOpen={setOpen}>
          <SidebarBody className="justify-between gap-10 bg-background">
            <div className="flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
              <div className="flex flex-col gap-1">
                {links.map((link) => (
                  <SidebarLink key={link.label} link={link} />
                ))}
              </div>
            </div>

            {/* Footer — logout */}
            <div
              onClick={handleLogout}
              className="cursor-pointer"
            >
              <SidebarLink
                link={{
                  label: t("auth.logout"),
                  href: "#",
                  icon: (
                    <LogOut className="h-5 w-5 flex-shrink-0" />
                  ),
                }}
              />
            </div>
          </SidebarBody>
        </Sidebar>

        <main className="flex-1 overflow-y-auto relative">
          <div
            className="fixed inset-0 pointer-events-none opacity-[0.15] dark:opacity-[0.15] z-0"
            style={{
              backgroundImage: "radial-gradient(circle, currentColor 1px, transparent 1px)",
              backgroundSize: "8px 8px",
              maskImage: "radial-gradient(ellipse at center, transparent 20%, black 80%)",
              WebkitMaskImage: "radial-gradient(ellipse at center, transparent 20%, black 80%)",
            }}
          />
          <div className="relative max-w-6xl mx-auto p-6 md:p-8">{children}</div>
        </main>
      </div>
      </div>
    </AuthGuard>
  );
}
