/** Placeholder contract for Task 0.5 runtime wiring. */
export interface PublicSummary {
  totalGroups: number;
  manualUpdate: true;
  updatedAt: string | null;
}

export interface PublicSummaryResponse {
  data: PublicSummary;
}
