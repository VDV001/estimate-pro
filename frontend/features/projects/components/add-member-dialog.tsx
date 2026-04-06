// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useTranslations } from "next-intl";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { addMember, searchUsers, listColleagues, listRecentlyAdded, type UserSearchResult } from "@/features/projects/api";

const ROLES = ["admin", "pm", "tech_lead", "developer", "observer"] as const;

interface AddMemberDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
}

export function AddMemberDialog({
  open,
  onOpenChange,
  projectId,
}: AddMemberDialogProps) {
  const t = useTranslations("projects");
  const tRoles = useTranslations("roles");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();

  const [query, setQuery] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<string>("");
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const debouncedQuery = useDebounce(query, 300);

  const { data: searchResults } = useQuery({
    queryKey: ["user-search", debouncedQuery],
    queryFn: () => searchUsers(debouncedQuery),
    enabled: debouncedQuery.length >= 2,
  });

  const { data: colleagues } = useQuery({
    queryKey: ["colleagues"],
    queryFn: listColleagues,
    enabled: open,
  });

  const { data: recentlyAdded } = useQuery({
    queryKey: ["recently-added"],
    queryFn: listRecentlyAdded,
    enabled: open,
  });

  const defaultSuggestions = (() => {
    const seen = new Set<string>();
    const merged: UserSearchResult[] = [];
    for (const list of [recentlyAdded ?? [], colleagues ?? []]) {
      for (const u of list) {
        if (!seen.has(u.id)) {
          seen.add(u.id);
          merged.push(u);
        }
      }
    }
    return merged;
  })();

  const suggestions = debouncedQuery.length >= 2
    ? (searchResults ?? [])
    : defaultSuggestions;

  const mutation = useMutation({
    mutationFn: () => addMember(projectId, { email, role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["members", projectId] });
      queryClient.invalidateQueries({ queryKey: ["recently-added"] });
      setQuery("");
      setEmail("");
      setRole("");
      onOpenChange(false);
    },
  });

  const selectUser = useCallback((user: UserSearchResult) => {
    setEmail(user.email);
    setQuery(`${user.name} (${user.email})`);
    setShowDropdown(false);
  }, []);

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!email.trim() || !role) return;
    mutation.mutate();
  };

  // Close dropdown on outside click
  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("inviteTitle")}</DialogTitle>
          <DialogDescription>{t("inviteDesc")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2 relative" ref={dropdownRef}>
            <Label htmlFor="member-search">{t("memberEmail")}</Label>
            <Input
              id="member-search"
              type="text"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setEmail("");
                setShowDropdown(true);
              }}
              onFocus={() => setShowDropdown(true)}
              placeholder={t("memberEmailPlaceholder")}
              autoComplete="off"
              autoFocus
            />
            {showDropdown && suggestions.length > 0 && (
              <div className="absolute z-50 top-full left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-md border bg-popover shadow-md">
                {suggestions.map((user) => (
                  <button
                    key={user.id}
                    type="button"
                    onClick={() => selectUser(user)}
                    className="flex flex-col w-full px-3 py-2 text-left hover:bg-muted transition-colors"
                  >
                    <span className="text-sm font-medium">{user.name}</span>
                    <span className="text-xs text-muted-foreground">{user.email}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="member-role">{t("memberRole")}</Label>
            <select
              id="member-role"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <option value="" disabled>
                {t("selectRole")}
              </option>
              {ROLES.map((r) => (
                <option key={r} value={r}>
                  {tRoles(r)}
                </option>
              ))}
            </select>
          </div>
          {mutation.isError && (
            <p className="text-sm text-destructive">
              {mutation.error?.message}
            </p>
          )}
          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {tCommon("cancel")}
            </Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !email.trim() || !role}
            >
              {mutation.isPending ? t("adding") : t("addMember")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function useDebounce(value: string, delay: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}
