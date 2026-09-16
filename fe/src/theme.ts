import { ref } from "vue";

export type Theme = "light" | "dark" | "system";

const THEME_KEY = "etle_theme";

export const currentTheme = ref<Theme>(
  (localStorage.getItem(THEME_KEY) as Theme) || "system"
);

function applyTheme(theme: Theme) {
  const isDark =
    theme === "dark" ||
    (theme === "system" &&
      window.matchMedia &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  if (isDark) {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
}

export function setTheme(theme: Theme) {
  currentTheme.value = theme;
  localStorage.setItem(THEME_KEY, theme);
  applyTheme(theme);
}

export function initTheme() {
  applyTheme(currentTheme.value);

  if (window.matchMedia) {
    window
      .matchMedia("(prefers-color-scheme: dark)")
      .addEventListener("change", () => {
        if (currentTheme.value === "system") {
          applyTheme("system");
        }
      });
  }
}
