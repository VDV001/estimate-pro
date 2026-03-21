// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  FolderKanban,
  FolderOpen,
  Archive,
  RotateCcw,
} from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import {
  listWorkspaces,
  listProjects,
  archiveProject,
  restoreProject,
} from "@/features/projects/api";
import { CreateProjectDialog } from "@/features/projects/components/create-project-dialog";

export default function ProjectsPage() {
  const t = useTranslations("projects");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);

  const { data: workspaces } = useQuery({
    queryKey: ["workspaces"],
    queryFn: listWorkspaces,
    retry: false,
  });

  const workspaceId = workspaces?.[0]?.id;

  const { data: projectsData, isLoading } = useQuery({
    queryKey: ["projects"],
    queryFn: () => listProjects(),
  });

  const projects = projectsData?.projects ?? [];
  const activeProjects = projects.filter((p) => p.status === "active");
  const archivedProjects = projects.filter((p) => p.status === "archived");

  const archiveMutation = useMutation({
    mutationFn: archiveProject,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["projects"] }),
  });

  const restoreMutation = useMutation({
    mutationFn: restoreProject,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["projects"] }),
  });

  const statusColor = (status: string) => {
    switch (status) {
      case "active":
        return "text-emerald-500";
      case "archived":
        return "text-muted-foreground";
      default:
        return "text-blue-500";
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24 text-muted-foreground">
        <p className="text-sm">{tCommon("loading")}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-2xl font-bold tracking-tight">{t("title")}</h1>
        <Button variant="outline" onClick={() => setCreateOpen(true)} className="gap-2">
          <Plus className="h-4 w-4" />
          {t("create")}
        </Button>
      </div>

      {/* Active Projects */}
      {activeProjects.length === 0 && archivedProjects.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <FolderKanban className="h-12 w-12 text-muted-foreground/30 mb-4" />
          <p className="text-lg font-medium mb-1">{t("empty")}</p>
          <p className="text-sm text-muted-foreground mb-8 text-center max-w-sm">
            {t("emptyDesc")}
          </p>
        </div>
      ) : (
        <>
          {activeProjects.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
              {activeProjects.map((project) => (
                <div
                  key={project.id}
                  className="rounded-xl border border-foreground/10 bg-background p-5 transition-colors duration-200 hover:bg-muted group"
                >
                  <Link
                    href={`/dashboard/projects/${project.id}`}
                    className="block mb-3"
                  >
                    <div className="inline-flex items-center justify-center h-9 w-9 rounded-lg bg-muted/50 mb-3">
                      <FolderOpen className="h-4 w-4 text-foreground" />
                    </div>
                    <h3 className="text-base font-semibold mb-1">
                      {project.name}
                    </h3>
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {project.description || "\u00A0"}
                    </p>
                  </Link>
                  <div className="flex items-center justify-between">
                    <span
                      className={`text-xs font-medium ${statusColor(project.status)}`}
                    >
                      {t("active")}
                    </span>
                    <button
                      onClick={() => archiveMutation.mutate(project.id)}
                      disabled={archiveMutation.isPending}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-foreground"
                      title={t("archive")}
                    >
                      <Archive className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Archived Projects */}
          {archivedProjects.length > 0 && (
            <>
              <h2 className="text-sm font-medium text-muted-foreground mb-4">
                {t("archived")} ({archivedProjects.length})
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {archivedProjects.map((project) => (
                  <div
                    key={project.id}
                    className="rounded-xl border border-foreground/5 bg-muted/30 p-5 group"
                  >
                    <Link
                      href={`/dashboard/projects/${project.id}`}
                      className="block mb-3"
                    >
                      <div className="inline-flex items-center justify-center h-9 w-9 rounded-lg bg-muted/50 mb-3">
                        <FolderOpen className="h-4 w-4 text-muted-foreground" />
                      </div>
                      <h3 className="text-base font-semibold text-muted-foreground mb-1">
                        {project.name}
                      </h3>
                      <p className="text-sm text-muted-foreground/70 line-clamp-2">
                        {project.description || "\u00A0"}
                      </p>
                    </Link>
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-muted-foreground">
                        {t("archived")}
                      </span>
                      <button
                        onClick={() => restoreMutation.mutate(project.id)}
                        disabled={restoreMutation.isPending}
                        className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-foreground"
                        title={t("restore")}
                      >
                        <RotateCcw className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </>
      )}

      <CreateProjectDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        workspaceId={workspaceId ?? ""}
      />
    </div>
  );
}
