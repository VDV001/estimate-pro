// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { UserPlus, Trash2, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { listMembers, removeMember } from "@/features/projects/api";
import type { MemberWithUser } from "@/features/projects/api";
import { AddMemberDialog } from "./add-member-dialog";

const ROLE_COLORS: Record<string, string> = {
  admin: "bg-violet-500/10 text-violet-500",
  pm: "bg-blue-500/10 text-blue-500",
  tech_lead: "bg-cyan-500/10 text-cyan-500",
  developer: "bg-emerald-500/10 text-emerald-500",
  observer: "bg-gray-500/10 text-gray-500",
};

function getInitial(name: string): string {
  return name.charAt(0).toUpperCase();
}

interface MembersListProps {
  projectId: string;
}

export function MembersList({ projectId }: MembersListProps) {
  const t = useTranslations("projects");
  const tRoles = useTranslations("roles");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();

  const [dialogOpen, setDialogOpen] = useState(false);

  const {
    data: members,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["members", projectId],
    queryFn: () => listMembers(projectId),
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) => removeMember(projectId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["members", projectId] });
    },
  });

  const handleRemove = (member: MemberWithUser) => {
    const message = t("confirmRemove", { name: member.user_name });
    if (window.confirm(message)) {
      removeMutation.mutate(member.user_id);
    }
  };

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

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold">{t("members")}</h3>
        <Button variant="outline" size="sm" onClick={() => setDialogOpen(true)} className="gap-2">
          <UserPlus className="h-4 w-4" />
          {t("addMember")}
        </Button>
      </div>

      {!members || members.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Users className="h-10 w-10 text-muted-foreground/30 mb-3" />
          <p className="text-sm font-medium text-muted-foreground">
            {t("noMembers")}
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            {t("noMembersDesc")}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {members.map((member) => (
            <div
              key={member.user_id}
              className="flex items-center justify-between rounded-lg border p-3"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                  {getInitial(member.user_name)}
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">
                    {member.user_name}
                  </p>
                  <p className="text-xs text-muted-foreground truncate">
                    {member.user_email}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${ROLE_COLORS[member.role] ?? ROLE_COLORS.observer}`}
                >
                  {tRoles(member.role as "admin" | "pm" | "tech_lead" | "developer" | "observer")}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-muted-foreground hover:text-destructive"
                  onClick={() => handleRemove(member)}
                  disabled={removeMutation.isPending}
                >
                  <Trash2 className="h-4 w-4" />
                  <span className="sr-only">{t("removeMember")}</span>
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <AddMemberDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        projectId={projectId}
      />
    </div>
  );
}
