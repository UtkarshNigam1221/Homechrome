import type { OpenNextConfig } from "@opennextjs/aws/types/open-next.js";

const config = {
  default: {
    override: {
      queue: "direct",
    },
  },
  dangerous: {
    disableTagCache: true,
  },
} satisfies OpenNextConfig;

export default config;
