package site

import "testing"

func TestBuildDeletePayload(t *testing.T) {
	payload, err := buildDeletePayload("42")
	if err != nil {
		t.Fatalf("buildDeletePayload: %v", err)
	}
	ids := payload["id__in"].([]int)
	if len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("bad payload %+v", payload)
	}
}

func TestBuildDeletePayloadRejectsNonNumericID(t *testing.T) {
	if _, err := buildDeletePayload("site-a"); err == nil {
		t.Fatalf("expected non-numeric id error")
	}
}
