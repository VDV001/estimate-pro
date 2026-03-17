"use client";

import { use, useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery } from "@tanstack/react-query";
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
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import { getProject } from "@/features/projects/api";
import { MembersList } from "@/features/projects/components/members-list";
import { DocumentsList } from "@/features/documents/components/documents-list";
import { Timeline } from "@/components/ui/timeline";
import type { TimelineEntry } from "@/components/ui/timeline";

type Tab = "overview" | "documents" | "members";

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

  const [activeTab, setActiveTab] = useState<Tab>("overview");

  const {
    data: project,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["project", id],
    queryFn: () => getProject(id),
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
      key: "members",
      label: t("tabs.members"),
      icon: <Users className="h-4 w-4" />,
    },
  ];

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link href="/dashboard">
          <Button variant="outline" size="icon">
            <ArrowLeft className="h-4 w-4" />
            <span className="sr-only">{tCommon("back")}</span>
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {project.name}
          </h1>
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
        <DocumentsList projectId={id} />
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
  project: { name: string; description: string; status: string; created_at: string };
}) {
  const t = useTranslations("projects");
  const tTimeline = useTranslations("projects.timeline");

  const createdDate = new Date(project.created_at).toLocaleDateString();

  const timelineData: TimelineEntry[] = [
    {
      title: tTimeline("created"),
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CalendarDays className="h-4 w-4" />
            <span>{createdDate}</span>
          </div>
          <p className="text-sm">{tTimeline("createdDesc")}</p>
        </div>
      ),
    },
    {
      title: tTimeline("team"),
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <UserPlus className="h-4 w-4" />
            <span>{tTimeline("teamDesc")}</span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("documents"),
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Upload className="h-4 w-4" />
            <span>{tTimeline("documentsDesc")}</span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("estimation"),
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="h-4 w-4" />
            <span>{tTimeline("estimationDesc")}</span>
          </div>
        </div>
      ),
    },
    {
      title: tTimeline("review"),
      content: (
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="h-4 w-4" />
            <span>{tTimeline("reviewDesc")}</span>
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

