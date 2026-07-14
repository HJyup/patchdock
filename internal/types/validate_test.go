package types

import "testing"

func validPlan() Plan {
	return Plan{
		TaskID:             "task-1",
		Approach:           "fix the bug in one file",
		AcceptanceCriteria: []string{"tests pass"},
		Steps:              []Step{{ID: "s1", Description: "edit the file"}},
	}
}

func TestNewPlanAcceptsValid(t *testing.T) {
	if _, err := NewPlan(validPlan()); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

func TestPlanDuplicateStepIDs(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{ID: "s1", Description: "first"},
		{ID: "s1", Description: "second"},
	}

	_, err := NewPlan(p)
	assertError(t, err, "plan.steps[1].id: duplicate of steps[0].id")
}

func TestPlanEmptyAcceptanceCriteria(t *testing.T) {
	p := validPlan()
	p.AcceptanceCriteria = nil

	_, err := NewPlan(p)
	assertError(t, err, "plan.acceptance_criteria: empty")
}

func TestReviewRejectRequiresIssues(t *testing.T) {
	_, err := NewReview(Review{
		TaskID:      "task-1",
		ExecutionID: "exec-1",
		Decision:    ReviewReject,
		Summary:     "does not compile",
	})
	assertError(t, err, "review.issues: required when decision is reject")
}

func TestReviewAcceptForbidsIssues(t *testing.T) {
	_, err := NewReview(Review{
		TaskID:      "task-1",
		ExecutionID: "exec-1",
		Decision:    ReviewAccept,
		Summary:     "looks good",
		Issues: []ReviewIssue{
			{Severity: SeverityMinor, Message: "nit"},
		},
	})
	assertError(t, err, "review.issues: must be empty when decision is accept")
}

func TestExecutionResultInvalidStatus(t *testing.T) {
	_, err := NewExecutionResult(ExecutionResult{
		TaskID: "task-1",
		PlanID: "plan-1",
		Status: "weird",
	})
	assertError(t, err, `execution_result.status: invalid value "weird"`)
}

func TestReviewJoinsAndSortsMultipleErrors(t *testing.T) {
	_, err := NewReview(Review{})
	assertError(t, err, "review.decision: empty\n"+
		"review.execution_id: empty\n"+
		"review.summary: empty\n"+
		"review.task_id: empty")
}

func assertError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if err.Error() != want {
		t.Fatalf("error mismatch\n got: %q\nwant: %q", err.Error(), want)
	}
}
