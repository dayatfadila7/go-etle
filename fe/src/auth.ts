import { createAuthClient } from "better-auth/client";

export const authClient = createAuthClient({
  baseURL: `${window.location.origin}/api/auth`,
});

export const { signIn, signOut, getSession, useSession } = authClient;
