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

  http.post(`${API}/auth/avatar`, () =>
    HttpResponse.json({ avatar_url: "http://localhost:9000/avatars/u1.jpg" }),
  ),

  http.post(`${API}/auth/forgot-password`, () =>
    HttpResponse.json({ message: "ok" }),
  ),

  http.post(`${API}/auth/reset-password`, () =>
    HttpResponse.json({ message: "ok" }),
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

  http.patch(`${API}/workspaces/:id`, ({ params }) =>
    HttpResponse.json({ id: params.id, name: "Renamed Workspace" }),
  ),

  // Estimation detail
  http.get(`${API}/projects/:id/estimations/:estId`, () =>
    HttpResponse.json({
      id: "e1",
      status: "draft",
      items: [
        { id: "i1", task_name: "Task 1", min_hours: 1, likely_hours: 2, max_hours: 4, sort_order: 0 },
      ],
    }),
  ),

  // User search
  http.get(`${API}/auth/search`, () =>
    HttpResponse.json([
      { id: "u3", name: "Found User", email: "found@test.com" },
    ]),
  ),

  http.get(`${API}/auth/colleagues`, () =>
    HttpResponse.json([
      { id: "u2", name: "Colleague", email: "colleague@test.com" },
    ]),
  ),

  http.get(`${API}/auth/recently-added`, () =>
    HttpResponse.json([
      { id: "u4", name: "Recent", email: "recent@test.com" },
    ]),
  ),

  // Extractions
  http.post(
    `${API}/projects/:projectId/documents/:docId/versions/:versionId/extractions`,
    async ({ request, params }) => {
      const body = (await request.json()) as { file_size?: number };
      return HttpResponse.json(
        {
          id: "ext-new",
          document_id: params.docId,
          document_version_id: params.versionId,
          status: "pending",
          tasks: [],
          created_at: "2026-05-05T00:00:00Z",
          updated_at: "2026-05-05T00:00:00Z",
          _echo_file_size: body.file_size,
        },
        { status: 201 },
      );
    },
  ),

  http.get(`${API}/extractions/:extractionId`, ({ params }) =>
    HttpResponse.json({
      extraction: {
        id: params.extractionId,
        document_id: "d1",
        document_version_id: "v1",
        status: "completed",
        tasks: [
          { name: "Implement login", estimate_hint: "small" },
          { name: "Wire OAuth" },
        ],
        created_at: "2026-05-05T00:00:00Z",
        updated_at: "2026-05-05T00:01:00Z",
        completed_at: "2026-05-05T00:01:00Z",
      },
      events: [
        {
          id: "ev1",
          extraction_id: params.extractionId,
          from_status: "pending",
          to_status: "processing",
          actor: "worker",
          created_at: "2026-05-05T00:00:30Z",
        },
      ],
    }),
  ),

  http.post(`${API}/extractions/:extractionId/cancel`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.post(`${API}/extractions/:extractionId/retry`, () =>
    new HttpResponse(null, { status: 204 }),
  ),

  http.get(`${API}/projects/:projectId/extractions`, () =>
    HttpResponse.json([
      {
        id: "ext1",
        document_id: "d1",
        document_version_id: "v1",
        status: "completed",
        tasks: [{ name: "Task A" }],
        created_at: "2026-05-05T00:00:00Z",
        updated_at: "2026-05-05T00:01:00Z",
      },
    ]),
  ),
];
