import express from "express";
import { toNodeHandler, fromNodeHeaders } from "better-auth/node";
import { auth, pgPool } from "./auth";

const app = express();

// blok sign-up publik: akun hanya boleh dibuat lewat seeder admin dari .env
app.all("/api/auth/sign-up/*", (_req, res) => {
  res.status(403).json({ error: "sign-up disabled" });
});

app.all("/api/auth/*", toNodeHandler(auth));
app.use(express.json());

const GO_API = process.env.GO_API_URL || "http://localhost:8080";

// semua panggilan data FE dilewatkan ke Go backend, HARUS sudah login (better-auth)
app.use("/api/data", async (req, res) => {
  const session = await auth.api.getSession({ headers: fromNodeHeaders(req.headers) });
  if (!session) {
    return res.status(401).json({ error: "unauthorized" });
  }
  const url = GO_API + req.originalUrl.replace(/^\/api\/data/, "");
  const body = ["GET", "HEAD"].includes(req.method) ? undefined : JSON.stringify(req.body);
  try {
    const upstream = await fetch(url, {
      method: req.method,
      headers: { "Content-Type": "application/json" },
      body,
    });
    const text = await upstream.text();
    res.status(upstream.status);
    const ct = upstream.headers.get("content-type");
    if (ct) res.setHeader("Content-Type", ct);
    res.send(text);
  } catch (e: any) {
    res.status(502).json({ error: e?.message || "bad gateway" });
  }
});

const PORT = Number(process.env.BFF_PORT || 3000);
app.listen(PORT, () => console.log(`[bff] better-auth + proxy on :${PORT}`));
