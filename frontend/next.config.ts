import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

const DEFAULT_GO_ORIGIN = "http://127.0.0.1:8300";

export default function nextConfig(phase: string): NextConfig {
  const isDevelopment = phase === PHASE_DEVELOPMENT_SERVER;

  return {
    output: "export",
    trailingSlash: true,
    skipTrailingSlashRedirect: true,
    ...(isDevelopment
      ? {
          async rewrites() {
            return [
              {
                source: "/api/:path*",
                destination: `${process.env.GO_API_ORIGIN ?? DEFAULT_GO_ORIGIN}/api/:path*`
              }
            ];
          }
        }
      : {})
  };
}
