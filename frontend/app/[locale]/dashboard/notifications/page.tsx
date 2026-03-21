// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useMemo } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { listProjects } from "@/features/projects/api";
import {
  listNotifications,
  markAllRead,
  type Notification,
} from "@/features/notifications/api";
import {
  ActivityLogsTable,
  type ActivityLog,
  type ActivityLevel,
} from "@/features/activity/components/activity-logs-table";

const eventLevelMap: Record<string, ActivityLevel> = {
  "member.added": "info",
  "document.uploaded": "success",
  "estimation.submitted": "info",
  "estimation.aggregated": "success",
};

export default function NotificationsPage() {
  const t = useTranslations();
  const queryClient = useQueryClient();

  const { data } = useQuery({
    queryKey: ["notifications", "list", 1],
    queryFn: () => listNotifications(1, 50),
  });

  const { data: projectsData } = useQuery({
    queryKey: ["projects"],
    queryFn: () => listProjects(),
  });

  const markAllReadMutation = useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const notifications = data?.notifications ?? [];
  const projects = projectsData?.projects ?? [];
  const hasUnread = notifications.some((n: Notification) => !n.read);

  const projectNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const p of projects) {
      map[p.id] = p.name;
    }
    return map;
  }, [projects]);

  const eventTypeLabel = (eventType: string): string => {
    const key = `notifications.eventType.${eventType.replace(".", "_")}`;
    return t(key);
  };

  const statusLabel = (read: boolean): string => {
    return read ? t("notifications.statusRead") : t("notifications.statusUnread");
  };

  const logs: ActivityLog[] = useMemo(() => {
    return notifications.map((n: Notification) => ({
      id: n.id,
      timestamp: n.created_at,
      level: eventLevelMap[n.event_type] ?? ("info" as ActivityLevel),
      service: n.project_id ? (projectNameMap[n.project_id] ?? t("notifications.unknownProject")) : t("notifications.noProject"),
      eventType: eventTypeLabel(n.event_type),
      message: n.message,
      status: statusLabel(n.read),
      tags: [eventTypeLabel(n.event_type)],
    }));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [notifications, projectNameMap]);

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-2xl font-bold tracking-tight">{t("notifications.title")}</h1>
        {hasUnread && (
          <Button
            variant="outline"
            size="sm"
            className="gap-2"
            onClick={() => markAllReadMutation.mutate()}
            disabled={markAllReadMutation.isPending}
          >
            <CheckCheck className="h-4 w-4" />
            {t("notifications.markAllRead")}
          </Button>
        )}
      </div>
      <ActivityLogsTable logs={logs} />
    </div>
  );
}
