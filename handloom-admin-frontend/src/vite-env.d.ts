/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_APP_ENV: string;
  readonly VITE_APP_NAME: string;
  readonly VITE_NEON_AUTH_URL: string;
  readonly VITE_NEON_DATA_API_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
