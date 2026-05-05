// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useTranslations } from "next-intl";
import { useQuery } from "@tanstack/react-query";
import { BarChart3 } from "lucide-react";
import { getAggregated, type AggregatedItem } from "@/features/estimation/api";
import { DownloadReportButton } from "@/features/report/components/download-report-button";

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface AggregatedViewProps {
  projectId: string;
}

export function AggregatedView({ projectId }: AggregatedViewProps) {
  const t = useTranslations("estimation");
  const tCommon = useTranslations("common");

  const { data, isLoading, isError } = useQuery({
    queryKey: ["estimations-aggregated", projectId],
    queryFn: () => getAggregated(projectId),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground">
        <p className="text-sm">{tCommon("loading")}</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center justify-center py-12 text-destructive">
        <p className="text-sm">{tCommon("error")}</p>
      </div>
    );
  }

  if (!data?.items || data.items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <BarChart3 className="h-10 w-10 text-muted-foreground/30 mb-3" />
        <p className="text-sm font-medium text-muted-foreground">
          {t("noAggregated")}
        </p>
        <p className="text-xs text-muted-foreground/70 mt-1">
          {t("noAggregatedDesc")}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">{t("aggregate")}</h3>
        <DownloadReportButton projectId={projectId} />
      </div>

      <div className="rounded-lg border">
        {/* Header */}
        <div className="grid grid-cols-[1fr_80px_80px_80px_72px] gap-2 px-4 py-2.5 border-b bg-muted/50 text-xs font-medium text-muted-foreground">
          <span>{t("task")}</span>
          <span className="text-right">{t("avgPert")}</span>
          <span className="text-right">{t("minOfMins")}</span>
          <span className="text-right">{t("maxOfMaxes")}</span>
          <span className="text-right">{t("estimators")}</span>
        </div>

        {/* Rows */}
        {data.items.map((item: AggregatedItem) => (
          <div
            key={item.task_name}
            className="grid grid-cols-[1fr_80px_80px_80px_72px] gap-2 px-4 py-2.5 border-b last:border-b-0 text-sm"
          >
            <span className="truncate font-medium">{item.task_name}</span>
            <span className="text-right tabular-nums font-semibold text-primary">
              {item.avg_pert_hours.toFixed(1)}
            </span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.min_of_mins}
            </span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.max_of_maxes}
            </span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.estimator_count}
            </span>
          </div>
        ))}

        {/* Total */}
        <div className="grid grid-cols-[1fr_80px_80px_80px_72px] gap-2 px-4 py-2.5 bg-muted/30 text-sm font-semibold">
          <span>{t("total")}</span>
          <span className="text-right tabular-nums text-primary">
            {data.total_hours.toFixed(1)}
          </span>
          <span />
          <span />
          <span />
        </div>
      </div>

      {/* PERT chart — visual bars */}
      <div className="space-y-2">
        {data.items!.map((item: AggregatedItem, idx: number) => {
          const maxWidth = Math.max(
            ...data.items!.map((i: AggregatedItem) => i.max_of_maxes)
          );
          const barWidth = maxWidth > 0
            ? (item.avg_pert_hours / maxWidth) * 100
            : 0;

          const barColors = [
            "#8b5cf6",
            "#3b82f6",
            "#10b981",
            "#f59e0b",
            "#f43f5e",
            "#06b6d4",
            "#ec4899",
            "#6366f1",
          ];
          const barColor = barColors[idx % barColors.length];

          return (
            <div key={item.task_name} className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <span className="truncate text-muted-foreground">{item.task_name}</span>
                <span className="tabular-nums font-medium">
                  {item.avg_pert_hours.toFixed(1)} {t("hours")}
                </span>
              </div>
              <div className="h-3 rounded-full overflow-hidden" style={{ backgroundColor: "rgba(128,128,128,0.2)" }}>
                <div
                  className="h-full rounded-full"
                  style={{ width: `${Math.max(barWidth, 2)}%`, backgroundColor: barColor }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
