import path from "node:path";
import { fileURLToPath } from "node:url";
import dotenv from "dotenv";
import { Pool } from "pg";
import { Kysely, PostgresDialect } from "kysely";
import { betterAuth } from "better-auth";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
dotenv.config({ path: path.resolve(__dirname, "../../.env") });

export const pgPool = new Pool({
  user: process.env.DB_USERNAME || "ursa",
  password: process.env.DB_PASSWORD || "",
  host: process.env.DB_HOST || "localhost",
  port: Number(process.env.DB_PORT || 5432),
  database: process.env.DB_NAME || "etle",
});

export const db = new Kysely({
  dialect: new PostgresDialect({ pool: pgPool }),
});

export const auth = betterAuth({
  database: { db, type: "postgres" },
  baseURL: process.env.BFF_URL || "http://localhost:3000",
  trustedOrigins: (process.env.TRUSTED_ORIGINS || "http://localhost:3000,http://localhost:5173,http://103.199.117.48")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean),
  secret: process.env.BETTER_AUTH_SECRET || "dev-secret-change-me",
  emailAndPassword: { enabled: true },
});
