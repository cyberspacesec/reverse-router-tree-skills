package generator

import "testing"

func TestEvaluateRestorationRate(t *testing.T) {
	spec := &Spec{
		Seed: 7,
		Resources: []*Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*Operation{{
				Method: "GET", Kind: OpGetOne, Repeat: 3,
				PathVar: &PathVarSpec{
					Pattern: "integer",
					Values:  []string{"123", "456", "789"},
				},
			}},
		}},
	}
	DerivePathVarExpectations(spec.Resources[0].Operations[0].PathVar, "users", 3)
	report, err := Evaluate(spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("还原率不达标: %s", report)
	}
	if report.Recall != 1 || report.Precision != 1 {
		t.Fatalf("report=%s", report)
	}
}
