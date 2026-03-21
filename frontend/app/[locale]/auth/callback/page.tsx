// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useEffect } from "react";
import { useRouter } from "@/i18n/navigation";
import { setTokens } from "@/lib/api-client";
import { useAuthStore } from "@/features/auth/store";

export default function OAuthCallbackPage() {
  const router = useRouter();
  const initialize = useAuthStore((s) => s.initialize);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const accessToken = params.get("access_token");
    const refreshToken = params.get("refresh_token");

    if (accessToken && refreshToken) {
      setTokens(accessToken, refreshToken);
      initialize().then(() => {
        router.push("/dashboard");
      });
    } else {
      router.push("/login");
    }
  }, [initialize, router]);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
    </div>
  );
}
