// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { create } from "zustand";
import { getAccessToken } from "@/lib/api-client";
import { getQueryClient } from "@/lib/query-client";
import {
  login,
  register,
  logout,
  getCurrentUser,
  type User,
  type LoginRequest,
  type RegisterRequest,
} from "./api";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  initialize: () => Promise<void>;
  loginUser: (data: LoginRequest) => Promise<void>;
  registerUser: (data: RegisterRequest) => Promise<void>;
  logoutUser: () => void;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  isAuthenticated: false,

  initialize: async () => {
    const token = getAccessToken();
    if (!token) {
      set({ user: null, isAuthenticated: false, isLoading: false });
      return;
    }

    // If already authenticated, skip re-fetch (prevents flash on locale change)
    const state = useAuthStore.getState();
    if (state.isAuthenticated && state.user) {
      set({ isLoading: false });
      return;
    }

    try {
      const user = await getCurrentUser();
      set({ user, isAuthenticated: true, isLoading: false });
    } catch {
      logout();
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  loginUser: async (data: LoginRequest) => {
    const res = await login(data);
    getQueryClient().clear();
    set({ user: res.user, isAuthenticated: true });
  },

  registerUser: async (data: RegisterRequest) => {
    const res = await register(data);
    getQueryClient().clear();
    set({ user: res.user, isAuthenticated: true });
  },

  logoutUser: () => {
    logout();
    getQueryClient().clear();
    set({ user: null, isAuthenticated: false });
  },

  setUser: (user: User) => {
    set({ user });
  },
}));
