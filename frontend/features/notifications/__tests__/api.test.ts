import { describe, it, expect, beforeEach } from "vitest";
import { getUnreadCount, listNotifications, markRead, markAllRead } from "../api";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("notifications API", () => {
  it("getUnreadCount returns count", async () => {
    const data = await getUnreadCount();
    expect(data.count).toBe(3);
  });

  it("listNotifications returns notifications", async () => {
    const data = await listNotifications(1, 50);
    expect(data.notifications).toHaveLength(1);
    expect(data.notifications![0].event_type).toBe("member.added");
  });

  it("markRead sends PATCH", async () => {
    await expect(markRead("n1")).resolves.toBeUndefined();
  });

  it("markAllRead sends PATCH", async () => {
    await expect(markAllRead()).resolves.toBeUndefined();
  });
});
