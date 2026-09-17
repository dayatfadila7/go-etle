async function req(path: string, opts: RequestInit = {}) {
  const res = await fetch("/api/data" + path, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  if (!res.ok) {
    const t = await res.text();
    throw new Error(t || res.statusText);
  }
  return res.json();
}

export const api = {
  dashboardStats: () => req("/dashboard/stats"),
  syncLogs: (limit = 30) => req("/sync-logs?limit=" + limit),
  syncAll: () => req("/sync-all", { method: "POST", body: "{}" }),
  sendPending: (limit = 200) => req(`/send?limit=${limit}`, { method: "POST", body: "{}" }),
  sendViolation: (id: number) => req(`/violations/${id}/send`, { method: "POST", body: "{}" }),
  cleanup: (days = 2) => req("/cleanup?days=" + days, { method: "POST", body: "{}" }),
  clients: () => req("/clients"),
  createClient: (b: any) => req("/clients", { method: "POST", body: JSON.stringify(b) }),
  updateClient: (id: number, b: any) => req(`/clients/${id}`, { method: "PUT", body: JSON.stringify({ ...b, id }) }),
  deleteClient: (id: number) => req(`/clients/${id}?id=${id}`, { method: "DELETE" }),
  syncMaster: (id: number) => req(`/clients/${id}/sync`, { method: "POST", body: "{}" }),
  master: () => req("/master"),
  violations: (status = "", page = 1, perPage = 20) =>
    req(`/violations?status=${status}&page=${page}&per_page=${perPage}`),
  violationDetail: (id: number | string) => req(`/violations/${id}`),
  cameras: (clientId = 0) => req("/cameras" + (clientId ? "?client_id=" + clientId : "")),
  createCamera: (b: any) => req("/cameras", { method: "POST", body: JSON.stringify(b) }),
  updateCamera: (id: number, b: any) => req(`/cameras/${id}`, { method: "PUT", body: JSON.stringify({ ...b, id }) }),
  deleteCamera: (id: number) => req(`/cameras/${id}?id=${id}`, { method: "DELETE" }),
  monitoring: () => req("/monitoring"),
};

