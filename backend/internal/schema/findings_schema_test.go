package schema_test

import "testing"

func TestFindingSchemaConstraintsAndRollback(t *testing.T) {
	pool := migrated(t)
	g := newGroup(t, pool, "Temuan")
	valid := `INSERT INTO evaluation_findings(group_id,file_type,location,problem,required_action,priority,created_at,updated_at) VALUES ($1,'PPT','Slide 5','Diagram salah','Perbaiki','WAJIB',now(),now())`
	mustExec(t, pool, valid, g)
	for _, sql := range []string{
		`UPDATE evaluation_findings SET file_type='EXE'`,
		`UPDATE evaluation_findings SET location=' '`,
		`UPDATE evaluation_findings SET problem=''`,
		`UPDATE evaluation_findings SET required_action=''`,
		`UPDATE evaluation_findings SET priority='TINGGI'`,
		`UPDATE study_groups SET review_status='SELESAI'`,
	} {
		mustFail(t, pool, "finding/review constraint", sql)
	}
	mustFail(t, pool, "findings protect group", `DELETE FROM study_groups WHERE id=$1`, g)
	goose(t, "down-to", "9")
	if tableExists(t, pool, "evaluation_findings") {
		t.Fatal("finding table survived down")
	}
	if !tableExists(t, pool, "submission_rounds") {
		t.Fatal("down removed older table")
	}
	goose(t, "up")
	if !tableExists(t, pool, "evaluation_findings") {
		t.Fatal("finding table missing after up")
	}
}
