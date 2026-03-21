// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, Globe, User, Check, Camera, Mail, MessageCircle, BellRing } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/hero-195-1";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { UserAvatar } from "@/components/ui/user-avatar";
import { useAuthStore } from "@/features/auth/store";
import { updateProfile, uploadAvatar } from "@/features/auth/api";
import { Switch } from "@/components/ui/switch";
import { getPreferences, setPreference, type NotificationPreference } from "@/features/notifications/api";

export default function SettingsPage() {
  const t = useTranslations();
  const user = useAuthStore((s) => s.user);
  const setUser = useAuthStore((s) => s.setUser);
  const [name, setName] = useState(user?.name ?? "");
  const [saved, setSaved] = useState(false);

  const mutation = useMutation({
    mutationFn: (newName: string) => updateProfile({ name: newName }),
    onSuccess: (updatedUser) => {
      setUser(updatedUser);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  const handleSave = () => {
    if (name.trim() && name !== user?.name) {
      mutation.mutate(name.trim());
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold tracking-tight mb-8 max-w-2xl mx-auto">
        {t("dashboard.settings")}
      </h1>

      <div className="space-y-6 max-w-2xl mx-auto">
        {/* Profile */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <User className="h-5 w-5 text-muted-foreground" />
              <CardTitle className="text-lg">{t("settings.profile")}</CardTitle>
            </div>
            <CardDescription>
              {t("settings.profileDesc")}
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-2">
            {/* Avatar with upload */}
            <div className="flex items-center gap-6 mb-5">
              <div className="relative group flex-shrink-0">
                <UserAvatar name={user?.name} avatarUrl={user?.avatar_url} size="lg" />
                <label className="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer">
                  <Camera className="h-6 w-6 text-white" />
                  <input
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    className="hidden"
                    onChange={async (e) => {
                      const file = e.target.files?.[0];
                      if (!file) return;
                      try {
                        const updatedUser = await uploadAvatar(file);
                        // Append timestamp to bust avatar cache
                        updatedUser.avatar_url = `${updatedUser.avatar_url}?t=${Date.now()}`;
                        setUser(updatedUser);
                      } catch {
                        // silently fail for now
                      }
                    }}
                  />
                </label>
              </div>
              <div className="space-y-1">
                <p className="text-xl font-semibold">{user?.name}</p>
                <p className="text-sm text-muted-foreground">{user?.email}</p>
                <p className="text-xs text-muted-foreground/50 mt-2">{t("settings.avatarHint")}</p>
              </div>
            </div>

            <div className="space-y-3 mt-6 mb-6">
              <Label>{t("auth.name")}</Label>
              <Input
                placeholder={t("settings.namePlaceholder")}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="flex items-center gap-3 mt-6">
              <Button
                onClick={handleSave}
                disabled={mutation.isPending || !name.trim() || name === user?.name}
              >
                {mutation.isPending ? t("common.loading") : t("common.save")}
              </Button>
              {saved && (
                <span className="inline-flex items-center gap-1 text-sm text-emerald-500">
                  <Check className="h-4 w-4" />
                </span>
              )}
              {mutation.isError && (
                <span className="text-sm text-destructive">{t("common.error")}</span>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Appearance */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Globe className="h-5 w-5 text-muted-foreground" />
              <CardTitle className="text-lg">{t("settings.appearance")}</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{t("settings.theme")}</p>
                <p className="text-sm text-muted-foreground">
                  {t("settings.themeDesc")}
                </p>
              </div>
              <ThemeToggle />
            </div>
          </CardContent>
        </Card>

        {/* Notifications */}
        <NotificationPreferences />
      </div>
    </div>
  );
}

function NotificationPreferences() {
  const t = useTranslations();
  const queryClient = useQueryClient();

  const { data } = useQuery({
    queryKey: ["notifications", "preferences"],
    queryFn: getPreferences,
  });

  const toggleMutation = useMutation({
    mutationFn: ({ channel, enabled }: { channel: string; enabled: boolean }) =>
      setPreference(channel, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications", "preferences"] });
    },
  });

  const prefs = data?.preferences ?? [];

  const channels = [
    {
      key: "in_app" as const,
      icon: <BellRing className="h-4 w-4" />,
      label: t("settings.channelInApp"),
      desc: t("settings.channelInAppDesc"),
      disabled: true,
    },
    {
      key: "email" as const,
      icon: <Mail className="h-4 w-4" />,
      label: t("settings.channelEmail"),
      desc: t("settings.channelEmailDesc"),
      disabled: false,
    },
    {
      key: "telegram" as const,
      icon: <MessageCircle className="h-4 w-4" />,
      label: t("settings.channelTelegram"),
      desc: t("settings.channelTelegramDesc"),
      disabled: false,
    },
  ];

  const isEnabled = (channel: string) => {
    const pref = prefs.find((p: NotificationPreference) => p.channel === channel);
    return pref?.enabled ?? channel === "in_app";
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Bell className="h-5 w-5 text-muted-foreground" />
          <CardTitle className="text-lg">
            {t("dashboard.notifications")}
          </CardTitle>
        </div>
        <CardDescription>
          {t("settings.notificationsDesc")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {channels.map((ch) => (
          <div key={ch.key} className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="text-muted-foreground">{ch.icon}</div>
              <div>
                <p className="text-sm font-medium">{ch.label}</p>
                <p className="text-xs text-muted-foreground">{ch.desc}</p>
              </div>
            </div>
            <Switch
              checked={isEnabled(ch.key)}
              onCheckedChange={(checked: boolean) => {
                if (!ch.disabled) {
                  toggleMutation.mutate({ channel: ch.key, enabled: checked });
                }
              }}
              disabled={ch.disabled || toggleMutation.isPending}
            />
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
