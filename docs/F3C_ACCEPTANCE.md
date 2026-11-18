# Fase 3C — Deployment and acceptance

## Release

User approved commit and production deployment on 18 November 2026.

- Backend: `edae2c8`; frontend: `6a7dfdd`; release source: `e4fbd337744f`.
- Production release: `20261118T111237Z-e4fbd337744f`.
- Migration: 9 → 10 (`evaluation_findings`).
- Pre-deploy database backup: `backups/pre-20261118T111237Z-e4fbd337744f.dump` (verified custom-format, 43,850 bytes).
- Attachment snapshot: same prefix, `.attachments` (0 files).
- Previous binary/public release retained: `20261118T045543Z-cffeaffbd4f3`.
- Production checkout was clean before deployment; update was fast-forward only. No dummy production data or account changes.

## Verification

Deployment script passed its pinned-source gates: sqlc generation, required database Go integration tests, vet, both Go builds, frontend lint/typecheck, 114 tests, and static build. Backup and migration succeeded; service readiness and smoke tests passed after restart.

Actual HTTPS checks at `https://fikriramaaulia.online`:

- Homepage serves the progress-first UI and anonymous progress API.
- API currently returns school year 2026/2027 with empty classes/groups. Browser displays the appropriate empty message.
- Admin login renders username/password fields; anonymous session check returns the expected 401.
- Edge headless at 360 CSS pixels: homepage and admin login have document width 360 and zero right-overflow elements.
- No observed CSP violations or runtime exceptions. Browser also requested a missing `/favicon.ico` (404); cosmetic, not changed in this batch.
- Internal readiness reports database OK; service active.

## Boundaries

Production contains no active public groups, so a populated group/correction workflow was **not** exercised in production. Original TypeScript client → real Go → test PostgreSQL integration passed in the isolated workspace: admin login, group, submissions, hidden draft, atomic publication with PERLU_REVISI, anonymous published read, DIPERIKSA transition, resolution, and anonymous resolved read. Exported-build component browser tests with local mock APIs covered unsaved-edit publication blocking, dirty new draft after confirmation, delayed and failed reload locking, and contextual accessible labels.

Single frontend review found four issues; user approved fixes, regression checks passed, and no review loop ran. Production authenticated click-through remains an operator acceptance step with real data. OpenAPI YAML parsed previously; full OpenAPI specification validation remains unavailable because `openapi_spec_validator` is not installed.

## Admin quick guide

1. Open `/admin/login/` and sign in. Visitors do not need accounts.
2. Register classes/groups in the active school year, then open a group detail.
3. Record PPT/PDF submissions and notebook receipt manually; this website does not read Gmail automatically.
4. Under **Catatan perbaikan**, enter task type, location, problem, requested correction, and priority. **Simpan draf** keeps it private.
5. Save/cancel any unfinished form edits. Select the intended review status, then **Terbitkan catatan dan status**, and confirm. All saved drafts become public together with the chosen status. Never put private data in published text.
6. Published wording is immutable. Check drafts carefully before publication. Published findings can be marked resolved.
7. To resolve the last mandatory open finding while the group is **Perlu Revisi**, first select and save **Sedang Diperiksa**, then mark the finding finished. Also use Sedang Diperiksa before correcting records blocked by an approved review state.
8. **Disetujui** requires the digital submissions and no open mandatory findings (including drafts). Overall completion additionally requires notebook receipt.
9. Check `/` anonymously: group progress and published correction notes appear together. `/progres/` remains available; announcements/materials are secondary menus.

## Rollback

Use the existing deployment rollback procedure in `docs/DEPLOYMENT.md`. Previous symlink targets remain available; migration rollback is not automatic. Restoring a database backup is destructive and requires a separate operator decision. Migration 10 adds a new table without dropping prior data.
