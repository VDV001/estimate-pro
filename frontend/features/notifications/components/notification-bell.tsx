// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell } from "lucide-react";
import {
  HoverCard,
  HoverCardTrigger,
  HoverCardContent,
} from "@/components/ui/hover-card";
import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import {
  getUnreadCount,
  listNotifications,
  markRead,
  markAllRead,
} from "@/features/notifications/api";
import { cn } from "@/lib/utils";

export function NotificationBell() {
  const t = useTranslations("notifications");
  const queryClient = useQueryClient();

  const { data: unreadData } = useQuery({
    queryKey: ["notifications", "unread-count"],
    queryFn: getUnreadCount,
    refetchInterval: 30000,
  });

  const { data: notifData } = useQuery({
    queryKey: ["notifications", "list", 1],
    queryFn: () => listNotifications(1, 5),
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => markRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const unreadCount = unreadData?.count ?? 0;
  const notifications = notifData?.notifications ?? [];

  const eventTypeLabel = (eventType: string) => {
    const key = `eventType.${eventType.replace(".", "_")}`;
    return t(key);
  };

  const formatTime = (dateStr: string) => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return t("justNow");
    if (mins < 60) return `${mins}m`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h`;
    const days = Math.floor(hours / 24);
    return `${days}d`;
  };

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button className="rounded-full p-1.5 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2">
          <Bell
            className={cn(
              "h-4 w-4 transition-colors",
              unreadCount > 0
                ? "text-destructive"
                : "text-muted-foreground hover:text-foreground"
            )}
          />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="end" className="w-80 p-0">
        {/* Header */}
        <div className="flex justify-between items-center px-4 py-3 border-b border-border">
          <p className="text-sm font-semibold">
            {t("title")}
            {unreadCount > 0 && (
              <span className="ml-2 text-xs font-normal text-destructive">
                {unreadCount}
              </span>
            )}
          </p>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="text-xs h-6 px-2"
              onClick={() => markAllReadMutation.mutate()}
            >
              {t("markAllRead")}
            </Button>
          )}
        </div>

        {/* List */}
        {notifications.length === 0 ? (
          <div className="p-5 text-center">
            <p className="text-sm text-muted-foreground">{t("empty")}</p>
            <p className="text-xs text-muted-foreground mt-1">{t("emptyDesc")}</p>
          </div>
        ) : (
          <div className="max-h-72 overflow-y-auto divide-y divide-border">
            {notifications.map((item) => (
              <button
                key={item.id}
                className={cn(
                  "w-full text-left px-4 py-3 hover:bg-muted/50 transition",
                  !item.read && "bg-primary/5"
                )}
                onClick={() => {
                  if (!item.read) markReadMutation.mutate(item.id);
                }}
              >
                <div className="flex justify-between items-center mb-0.5">
                  <span className={cn("text-sm", !item.read && "font-medium")}>
                    {eventTypeLabel(item.event_type)}
                  </span>
                  <span className="text-xs text-muted-foreground ml-2 flex-shrink-0">
                    {formatTime(item.created_at)}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  {item.message}
                </p>
              </button>
            ))}
          </div>
        )}

        {/* Footer */}
        <div className="border-t border-border px-4 py-2">
          <Link
            href="/dashboard/notifications"
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {t("viewAll")}
          </Link>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}
