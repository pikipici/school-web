export function formatWIB(value: string): string { return new Intl.DateTimeFormat("id-ID", { dateStyle: "long", timeStyle: "short", timeZone: "Asia/Jakarta" }).format(new Date(value)) + " WIB"; }
export function validContentID(value: string | null): number | null { if (!value || !/^[1-9]\d*$/.test(value)) return null; const id = Number(value); return Number.isSafeInteger(id) ? id : null; }
