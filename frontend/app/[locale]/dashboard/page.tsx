// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useMemo } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useQueries } from "@tanstack/react-query";
import { useAuthStore } from "@/features/auth/store";
import {
  Plus,
  FolderKanban,
  Users,
  FileText,
  BarChart3,
  Sparkles,
  LayoutGrid,
} from "lucide-react";
import { Link } from "@/i18n/navigation";
import { listWorkspaces, listProjects, listMembers, type Workspace } from "@/features/projects/api";
import { listDocuments } from "@/features/documents/api";
import { listEstimations, getAggregated, type Estimation, type AggregatedResult } from "@/features/estimation/api";

// ---------------------------------------------------------------------------
// Main Page
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const t = useTranslations();
  const user = useAuthStore((s) => s.user);

  const greeting = user?.name
    ? t("dashboard.welcomeBack", { name: user.name })
    : t("dashboard.welcome");

  // ---- Data fetching ----

  const { data: workspaces } = useQuery({
    queryKey: ["workspaces"],
    queryFn: listWorkspaces,
    retry: false,
  });

  const { data: projectsData } = useQuery({
    queryKey: ["projects"],
    queryFn: () => listProjects(),
  });

  const projects = projectsData?.projects ?? [];
  const projectCount = projectsData?.meta?.total ?? 0;

  const memberQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: ["members", p.id],
      queryFn: () => listMembers(p.id),
    })),
  });

  const docQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: ["documents", p.id],
      queryFn: () => listDocuments(p.id),
    })),
  });

  const estQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: ["estimations", p.id],
      queryFn: () => listEstimations(p.id),
    })),
  });

  const aggQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: ["aggregated", p.id],
      queryFn: () => getAggregated(p.id),
    })),
  });

  // ---- Computed stats ----

  const totalMembers = useMemo(
    () => new Set(memberQueries.flatMap((q) => (q.data ?? []).map((m) => m.user_id))).size,
    [memberQueries],
  );
  const totalDocs = useMemo(
    () => docQueries.reduce((sum, q) => sum + (q.data?.length ?? 0), 0),
    [docQueries],
  );
  const totalEstimations = useMemo(
    () => estQueries.reduce((sum, q) => sum + (q.data?.length ?? 0), 0),
    [estQueries],
  );



  // Estimation bars per project
  const estimationBars = useMemo(() => {
    return projects
      .map((p, idx) => {
        const agg: AggregatedResult | undefined = aggQueries[idx]?.data;
        return {
          name: p.name,
          totalHours: agg?.total_hours ?? 0,
          itemCount: agg?.items?.length ?? 0,
        };
      })
      .filter((b) => b.totalHours > 0);
  }, [projects, aggQueries]);

  const maxEstHours = Math.max(...estimationBars.map((b) => b.totalHours), 1);
  const totalEstHours = estimationBars.reduce((sum, b) => sum + b.totalHours, 0);

  // Chart bar colors (green for estimation bars)
  const barColors = ["#34d399", "#60a5fa", "#a78bfa", "#fbbf24", "#f472b6", "#22d3ee", "#4ade80"];
  // Timeline step colors (matching project timeline: blue→indigo→teal→emerald→green)
  const stepColors = ["#3b82f6", "#6366f1", "#14b8a6", "#10b981", "#22c55e"];

  // ---- Stats config ----

  const stats = [
    { key: "workspaces", icon: LayoutGrid, value: String(workspaces?.length ?? 0), label: t("dashboard.overview"), color: "text-pink-500", bg: "bg-pink-500/10" },
    { key: "projects", icon: FolderKanban, value: String(projectCount), label: t("projects.title"), color: "text-violet-500", bg: "bg-violet-500/10" },
    { key: "members", icon: Users, value: String(totalMembers), label: t("projects.members"), color: "text-blue-500", bg: "bg-blue-500/10" },
    { key: "documents", icon: FileText, value: String(totalDocs), label: t("projects.documents"), color: "text-emerald-500", bg: "bg-emerald-500/10" },
    { key: "estimations", icon: BarChart3, value: String(totalEstimations), label: t("projects.estimations"), color: "text-amber-500", bg: "bg-amber-500/10" },
  ];

  return (
    <div>
      {/* Welcome */}
      <div className="mb-10">
        <div className="flex items-center gap-2 mb-1">
          <Sparkles className="h-5 w-5 text-amber-500" />
          <h1 className="text-2xl font-bold tracking-tight">{greeting}</h1>
        </div>
        <p className="text-muted-foreground">{t("dashboard.welcomeSubtitle")}</p>
      </div>

      {/* Stats — clickable → drawer */}
      <div className="grid gap-2 mb-12" style={{ gridTemplateColumns: "repeat(5, 1fr)" }}>
        {stats.map((stat) => (
          <div
            key={stat.key}
            className="rounded-xl py-5 text-center flex flex-col items-center"
          >
            <div className={`inline-flex items-center justify-center h-9 w-9 rounded-lg ${stat.bg} mb-3`}>
              <stat.icon className={`h-5 w-5 ${stat.color}`} />
            </div>
            <p className="text-2xl font-bold">{stat.value}</p>
            <p className="text-sm text-muted-foreground">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* Workspace cards */}
      <div className="mb-12">
        <h2 className="text-lg font-semibold mb-4">{t("dashboard.overview")}</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {(workspaces ?? []).map((ws: Workspace) => (
              <Link
                key={ws.id}
                href="/dashboard/projects"
                className="rounded-2xl border border-foreground/10 bg-background p-6 cursor-pointer transition-colors duration-200 hover:bg-muted block"
              >
                <div className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-muted/50 mb-4">
                  <LayoutGrid className="h-5 w-5 text-foreground" />
                </div>
                <h3 className="text-lg font-semibold mb-2">{ws.name}</h3>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  <span className="inline-flex items-center gap-1">
                    <FolderKanban className="h-3.5 w-3.5" />
                    {t("dashboard.workspaceProjects", { count: projectCount })}
                  </span>
                </div>
              </Link>
          ))}

          {/* Create workspace card */}
          <div
            className="relative rounded-2xl cursor-not-allowed group"
            title={t("dashboard.workspaceComingSoon")}
          >
            <div className="relative rounded-2xl border border-dashed border-foreground/20 bg-background p-6 h-full flex flex-col items-center justify-center opacity-50">
              <div className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-muted/50 mb-4">
                <Plus className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-sm text-muted-foreground">{t("dashboard.createWorkspace")}</p>
              <p className="text-xs text-muted-foreground/60 mt-1">{t("dashboard.workspaceComingSoon")}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Estimation chart — vertical bars */}
      {estimationBars.length > 0 && (
        <div className="mb-12">
          <h2 className="text-lg font-semibold mb-4">{t("dashboard.estimationsByProject")}</h2>
          <div className="rounded-xl border border-foreground/10 bg-background p-6">
            <div className="flex justify-end mb-4">
              <p className="text-3xl font-bold tabular-nums tracking-tight">
                {totalEstHours.toFixed(1)}{t("estimation.hours")}
              </p>
            </div>
            <div className="flex items-end gap-3" style={{ height: "160px" }}>
              {estimationBars.map((bar, idx) => {
                const pct = maxEstHours > 0 ? (bar.totalHours / maxEstHours) * 100 : 0;
                const barHeight = Math.max(pct, 8);
                return (
                  <div key={bar.name} className="flex flex-col items-center justify-end gap-1.5" style={{ height: "100%", flex: 1, minWidth: 0 }}>
                    <span className="text-xs tabular-nums font-medium">{bar.totalHours.toFixed(1)}{t("estimation.hours")}</span>
                    <div
                      className="w-full max-w-16 rounded-md"
                      style={{
                        height: `${barHeight}%`,
                        backgroundColor: barColors[idx % barColors.length],
                        minHeight: "12px",
                      }}
                    />
                    <span className="text-xs text-muted-foreground truncate max-w-full text-center">{bar.name}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {/* Project progress */}
      {projects.length > 0 && (
        <div className="mb-12">
          <h2 className="text-lg font-semibold mb-4">{t("dashboard.projectProgress")}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {projects.map((p, idx) => {
              const mCount = memberQueries[idx]?.data?.length ?? 0;
              const dCount = docQueries[idx]?.data?.length ?? 0;
              const eSubmitted = (estQueries[idx]?.data ?? []).filter((e: Estimation) => e.status === "submitted").length;
              const hasAgg = (aggQueries[idx]?.data?.items?.length ?? 0) > 0;

              const steps = [true, mCount > 1, dCount > 0, eSubmitted > 0, hasAgg];
              const completed = steps.filter(Boolean).length;

              return (
                <Link
                  key={p.id}
                  href={`/dashboard/projects/${p.id}`}
                  className="rounded-xl border border-foreground/10 bg-background p-4 flex items-center gap-4 hover:bg-muted transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{p.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {completed} {t("dashboard.of")} {steps.length} {t("dashboard.step")}
                    </p>
                  </div>
                  {/* Mini progress bar */}
                  <div className="flex gap-1">
                    {steps.map((done, si) => (
                      <div
                        key={si}
                        className="h-2 w-5 rounded-full"
                        style={{
                          backgroundColor: done ? stepColors[si] : "rgba(128,128,128,0.2)",
                        }}
                      />
                    ))}
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      )}


    </div>
  );
}
