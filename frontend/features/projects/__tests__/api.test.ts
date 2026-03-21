import { describe, it, expect, beforeEach } from "vitest";
import { listProjects, createProject, listMembers, addMember, removeMember } from "../api";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("projects API", () => {
  it("listProjects returns projects with meta", async () => {
    const data = await listProjects();
    expect(data.projects).toHaveLength(1);
    expect(data.projects![0].name).toBe("Project 1");
    expect(data.meta.total).toBe(1);
  });

  it("createProject sends POST and returns new project", async () => {
    const project = await createProject({
      workspace_id: "w1",
      name: "New Project",
      description: "Description",
    });
    expect(project.id).toBe("p-new");
    expect(project.name).toBe("New Project");
  });

  it("listMembers returns members array", async () => {
    const members = await listMembers("p1");
    expect(members).toHaveLength(2);
    expect(members[0].role).toBe("admin");
    expect(members[1].role).toBe("developer");
  });

  it("addMember sends POST with email and role", async () => {
    const result = await addMember("p1", { email: "new@test.com", role: "developer" });
    expect(result.status).toBe("ok");
  });

  it("removeMember sends DELETE", async () => {
    await expect(removeMember("p1", "u2")).resolves.toBeUndefined();
  });
});
