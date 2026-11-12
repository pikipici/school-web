import type { PublicSummary, PublicSummaryResponse } from "@/types/public";

export const PUBLIC_SUMMARY_PATH = "/api/v1/public/summary" as const;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

function isPublicSummary(value: unknown): value is PublicSummary {
  if (!isRecord(value)) {
    return false;
  }

  const { totalGroups, manualUpdate, updatedAt } = value;
  const hasValidUpdatedAt =
    updatedAt === null ||
    (typeof updatedAt === "string" &&
      updatedAt.trim() !== "" &&
      !Number.isNaN(Date.parse(updatedAt)));

  return (
    typeof totalGroups === "number" &&
    Number.isSafeInteger(totalGroups) &&
    totalGroups >= 0 &&
    manualUpdate === true &&
    hasValidUpdatedAt
  );
}

function isPublicSummaryResponse(value: unknown): value is PublicSummaryResponse {
  return isRecord(value) && isPublicSummary(value.data);
}

export async function getPublicSummary(signal?: AbortSignal): Promise<PublicSummary> {
  const response = await fetch(PUBLIC_SUMMARY_PATH, {
    method: "GET",
    headers: { Accept: "application/json" },
    cache: "no-store",
    signal
  });

  if (!response.ok) {
    throw new Error("Public summary request failed");
  }

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    throw new Error("Public summary response was not JSON");
  }

  if (!isPublicSummaryResponse(payload)) {
    throw new Error("Public summary response did not match the contract");
  }

  return payload.data;
}
