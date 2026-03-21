import { http, HttpResponse } from "msw";

const API = "http://localhost:8080/api/v1";

export const handlers = [
  // Auth
  http.post(`${API}/auth/login`, () =>
    HttpResponse.json({
      user: { id: "u1", name: "Test User", email: "test@test.com" },
      access_token: "test-access-token",
      refresh_token: "test-refresh-token",
    }),
  ),

  http.post(`${API}/auth/register`, () =>
    HttpResponse.json({
      user: { id: "u1", name: "Test User", email: "test@test.com" },
      access_token: "test-access-token",
      refresh_token: "test-refresh-token",
    }),
  ),

  http.get(`${API}/auth/me`, () =>
    HttpResponse.json({ id: "u1", name: "Test User", email: "test@test.com" }),
  ),

  http.post(`${API}/auth/refresh`, () =>
    HttpResponse.json({
      access_token: "new-access-token",
      refresh_token: "new-refresh-token",
    }),
  ),

  http.patch(`${API}/auth/profile`, () =>
    HttpResponse.json({ id: "u1", name: "Updated Name", email: "test@test.com" }),
  ),

  // Projects
  http.get(`${API}/projects`, () =>
    HttpResponse.json({
      projects: [
        { id: "p1", name: "Project 1", description: "Desc 1", status: "active", created_at: "2026-01-01T00:00:00Z" },
      ],
      meta: { total: 1, page: 1, limit: 20 },
    }),
  ),

  http.post(`${API}/projects`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({
      id: "p-new",
      name: body.name,
      description: body.description,
      status: "active",
      created_at: "2026-03-21T00:00:00Z",
    });
  }),

  http.get(`${API}/projects/:id`, ({ params }) =>
    HttpResponse.json({
      id: params.id,
      name: "Project 1",
      description: "Desc",
      status: "active",
      created_at: "2026-01-01T00:00:00Z",
    }),
  ),

  // Members
  http.get(`${API}/projects/:id/members`, () =>
    HttpResponse.json([
      { user_id: "u1", role: "admin", user: { id: "u1", name: "Admin", email: "admin@test.com" } },
      { user_id: "u2", role: "developer", user: { id: "u2", name: "Dev", email: "dev@test.com" } },
    ]),
  ),

  http.post(`${API}/projects/:id/members`, () =>
    HttpResponse.json({ status: "ok" }),
  ),

  http.delete(`${API}/projects/:id/members/:userId`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  // Documents
  http.get(`${API}/projects/:id/documents`, () =>
    HttpResponse.json([
      { id: "d1", title: "spec.pdf", created_at: "2026-01-01T00:00:00Z" },
    ]),
  ),

  http.get(`${API}/projects/:projectId/documents/:docId`, () =>
    HttpResponse.json({
      id: "d1",
      title: "spec.pdf",
      created_at: "2026-01-01T00:00:00Z",
      latest_version: { id: "v1", is_signed: false, is_final: false, tags: [] },
    }),
  ),

  http.post(`${API}/projects/:id/documents`, () =>
    HttpResponse.json({ id: "d-new", title: "uploaded.pdf", version: { id: "v1" } }),
  ),

  http.delete(`${API}/projects/:projectId/documents/:docId`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.patch(`${API}/projects/:projectId/documents/:docId/versions/:versionId/flags`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.put(`${API}/projects/:projectId/documents/:docId/versions/:versionId/tags`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  // Estimations
  http.get(`${API}/projects/:id/estimations`, () =>
    HttpResponse.json([
      { id: "e1", status: "draft", created_at: "2026-01-01T00:00:00Z" },
    ]),
  ),

  http.post(`${API}/projects/:id/estimations`, () =>
    HttpResponse.json({
      id: "e-new",
      status: "draft",
      items: [{ id: "i1", task: "Task 1", min_hours: 1, likely_hours: 2, max_hours: 3 }],
    }),
  ),

  http.get(`${API}/projects/:id/estimations/aggregated`, () =>
    HttpResponse.json({
      total_hours: 100,
      items: [
        { task: "Task 1", avg_pert: 10, min_hours: 5, max_hours: 20, estimator_count: 3 },
      ],
    }),
  ),

  http.put(`${API}/projects/:id/estimations/:estId/submit`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.delete(`${API}/projects/:id/estimations/:estId`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  // Notifications
  http.get(`${API}/notifications`, () =>
    HttpResponse.json({
      notifications: [
        { id: "n1", event_type: "member.added", message: "Dani added to project", read: false, created_at: "2026-03-21T10:00:00Z" },
      ],
      meta: { total: 1, page: 1 },
    }),
  ),

  http.get(`${API}/notifications/unread-count`, () =>
    HttpResponse.json({ count: 3 }),
  ),

  http.patch(`${API}/notifications/:id/read`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.patch(`${API}/notifications/read-all`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.get(`${API}/notifications/preferences`, () =>
    HttpResponse.json({
      preferences: [
        { channel: "in_app", enabled: true },
        { channel: "email", enabled: false },
        { channel: "telegram", enabled: false },
      ],
    }),
  ),

  http.put(`${API}/notifications/preferences`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  // Workspaces
  http.get(`${API}/workspaces`, () =>
    HttpResponse.json([
      { id: "w1", name: "Default Workspace" },
    ]),
  ),
];
