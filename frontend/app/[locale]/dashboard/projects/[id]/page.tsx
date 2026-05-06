// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { use, useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { downloadReport, REPORT_FORMATS, type ReportFormat } from "@/features/report/api";
import {
  ArrowLeft,
  FileText,
  Users,
  LayoutDashboard,
  CalendarDays,
  UserPlus,
  Upload,
  CheckCircle2,
  Clock,
  Calculator,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import { getProject, listMembers, updateProject } from "@/features/projects/api";
import { InlineEdit } from "@/components/ui/inline-edit";
import { listDocuments } from "@/features/documents/api";
import { listEstimations, getAggregated } from "@/features/estimation/api";
import { MembersList } from "@/features/projects/components/members-list";
import { DocumentsList } from "@/features/documents/components/documents-list";
import { EstimationTab } from "@/features/estimation/components/estimation-tab";
import { Timeline } from "@/components/ui/timeline";
import type { TimelineEntry } from "@/components/ui/timeline";

type Tab = "overview" | "documents" | "estimations" | "members";

const STATUS_COLORS: Record<string, string> = {
  active: "bg-green-500/10 text-green-500",
  archived: "bg-gray-500/10 text-gray-500",
  in_review: "bg-yellow-500/10 text-yellow-500",
  planning: "bg-blue-500/10 text-blue-500",
};

export default function ProjectDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const t = useTranslations("projects");
  const tCommon = useTranslations("common");

  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const isDeeplink = searchParams.get("download") === "report";
  const [activeTab, setActiveTab] = useState<Tab>(isDeeplink ? "estimations" : "overview");
  const [pendingEstimationTasks, setPendingEstimationTasks] = useState<
    string[] | undefined
  >(undefined);

  const handleCreateEstimation = (taskNames: string[]) => {
    setPendingEstimationTasks(taskNames);
    setActiveTab("estimations");
  };

  const autoDownloadFiredRef = useRef(false);
  useEffect(() => {
    if (autoDownloadFiredRef.current) return;
    if (!isDeeplink) return;
    const format = searchParams.get("format") ?? "pdf";
    if (!REPORT_FORMATS.includes(format as ReportFormat)) return;
    autoDownloadFiredRef.current = true;
    void downloadReport(id, format as ReportFormat);
  }, [isDeeplink, searchParams, id]);

  const {
    data: project,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["project", id],
    queryFn: () => getProject(id),
  });

  const renameMutation = useMutation({
    mutationFn: (name: string) => updateProject(id, { name }),
    onSuccess: (updated) => {
      queryClient.setQueryData(["project", id], updated);
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24 text-muted-foreground">
        <p className="text-sm">{tCommon("loading")}</p>
      </div>
    );
  }

  if (isError || !project) {
    return (
      <div className="flex items-center justify-center py-24 text-destructive">
        <p className="text-sm">{tCommon("error")}</p>
      </div>
    );
  }

  const statusKey =
    project.status === "active"
      ? "active"
      : project.status === "archived"
        ? "archived"
        : project.status;

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    {
      key: "overview",
      label: t("tabs.overview"),
      icon: <LayoutDashboard className="h-4 w-4" />,
    },
    {
      key: "documents",
      label: t("tabs.documents"),
      icon: <FileText className="h-4 w-4" />,
    },
    {
      key: "estimations",
      label: t("tabs.estimations"),
      icon: <Calculator className="h-4 w-4" />,
    },
    {
      key: "members",
      label: t("tabs.members"),
      icon: <Users className="h-4 w-4" />,
    },
  ];

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link href="/dashboard/projects">
          <Button variant="outline" size="icon">
            <ArrowLeft className="h-4 w-4" />
            <span className="sr-only">{tCommon("back")}</span>
          </Button>
        </Link>
        <div>
          <InlineEdit
            value={project.name}
            onSave={(name) => renameMutation.mutate(name)}
            className="text-2xl font-bold tracking-tight"
            inputClassName="text-2xl font-bold h-10"
          />
          <div className="flex items-center gap-2 mt-1">
            <span
              className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_COLORS[project.status] ?? STATUS_COLORS.active}`}
            >
              {t(statusKey as "active" | "archived")}
            </span>
          </div>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-1 border-b mb-6">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.key
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/30"
            }`}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && (
        <OverviewTab project={project} />
      )}
      {activeTab === "documents" && (
        <DocumentsList
          projectId={id}
          onCreateEstimation={handleCreateEstimation}
        />
      )}
      {activeTab === "estimations" && (
        <EstimationTab
          key={pendingEstimationTasks?.join("|") ?? "default"}
          projectId={id}
          initialTasks={pendingEstimationTasks}
          onTasksConsumed={() => setPendingEstimationTasks(undefined)}
        />
      )}
      {activeTab === "members" && (
        <MembersList projectId={id} />
      )}
    </div>
  );
}

function OverviewTab({
  project,
}: {
  project: { id: string; name: string; description: string; status: string; created_at: string };
}) {
  const t = useTranslations("projects");
  const tTimeline = useTranslations("projects.timeline");

  const createdDate = new Date(project.created_at).toLocaleDateString();

  const { data: members } = useQuery({
    queryKey: ["members", project.id],
    queryFn: () => listMembers(project.id),
  });

  const { data: documents } = useQuery({
    queryKey: ["documents", project.id],
    queryFn: () => listDocuments(project.id),
  });

  const { data: estimations } = useQuery({
    queryKey: ["estimations", project.id, "all"],
    queryFn: () => listEstimations(project.id),
  });

  const { data: aggregated } = useQuery({
    queryKey: ["aggregated", project.id],
    queryFn: () => getAggregated(project.id),
  });

  const membersCount = members?.length ?? 0;
  const docsCount = documents?.length ?? 0;
  const submittedCount = estimations?.filter((e) => e.status === "submitted").length ?? 0;
  const hasAggregated = (aggregated?.items?.length ?? 0) > 0;

  const hasTeam = membersCount > 1;
  const hasDocs = docsCount > 0;
  const hasEstimations = submittedCount > 0;

  const timelineData: TimelineEntry[] = [
    {
      title: tTimeline("created"),
      completed: true,
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CalendarDays className="h-4 w-4" />
            <span>{tTimeline("createdDone", { date: createdDate })}</span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("team"),
      completed: hasTeam,
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <UserPlus className="h-4 w-4" />
            <span>
              {hasTeam
                ? tTimeline("teamDone", { count: membersCount })
                : tTimeline("teamDesc")}
            </span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("documents"),
      completed: hasDocs,
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Upload className="h-4 w-4" />
            <span>
              {hasDocs
                ? tTimeline("documentsDone", { count: docsCount })
                : tTimeline("documentsDesc")}
            </span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("estimation"),
      completed: hasEstimations,
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="h-4 w-4" />
            <span>
              {hasEstimations
                ? tTimeline("estimationDone", { count: submittedCount })
                : tTimeline("estimationDesc")}
            </span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("review"),
      completed: hasAggregated,
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4" />
            <span>
              {hasAggregated
                ? tTimeline("reviewDone", { hours: aggregated!.total_hours.toFixed(1) })
                : tTimeline("reviewDesc")}
            </span>
          </div>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Project Info Card */}
      <div className="rounded-lg border p-6 space-y-4">
        <div>
          <p className="text-sm font-medium text-muted-foreground">
            {t("name")}
          </p>
          <p className="text-base mt-1">{project.name}</p>
        </div>
        <div>
          <p className="text-sm font-medium text-muted-foreground">
            {t("description")}
          </p>
          <p className="text-base mt-1">
            {project.description || "\u2014"}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="text-sm font-medium text-muted-foreground">
              {t("status")}
            </p>
            <p className="text-base mt-1">
              {t(
                project.status === "active"
                  ? "active"
                  : project.status === "archived"
                    ? "archived"
                    : "active"
              )}
            </p>
          </div>
          <div>
            <p className="text-sm font-medium text-muted-foreground">
              {t("createdAt")}
            </p>
            <p className="text-base mt-1">
              {new Date(project.created_at).toLocaleDateString()}
            </p>
          </div>
        </div>
      </div>

      {/* Project Timeline */}
      <Timeline data={timelineData} />
    </div>
  );
}

