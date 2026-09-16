import { auth } from "./auth";

async function main() {
  const email = process.env.ADMIN_EMAIL;
  const password = process.env.ADMIN_PASSWORD;
  if (!email || !password) {
    console.error("ADMIN_EMAIL / ADMIN_PASSWORD belum di-set di .env");
    process.exit(1);
  }

  // pastikan DB & better-auth sudah siap
  await new Promise((r) => setTimeout(r, 500));

  try {
    const res: any = await auth.api.signUpEmail({
      body: { email, password, name: process.env.ADMIN_NAME || "Admin" },
    });
    console.log("admin dibuat:", res?.user?.email || email);
  } catch (e: any) {
    const msg = String(e?.message || e?.statusText || e);
    if (/already|exists|sudah/i.test(msg)) {
      console.log("admin sudah ada:", email);
    } else {
      console.error("gagal buat admin:", msg);
      process.exit(1);
    }
  }
  process.exit(0);
}

main();
