import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  transpilePackages: ["@seatd/typescript-seatd-client"],
  typescript: {
    ignoreBuildErrors: true,
  },
};

export default nextConfig;
