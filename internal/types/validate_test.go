package types

import "testing"

func validPlan() Plan {
	return Plan{
		TaskID:  "task-1",
		Summary: "fix the bug in one file",
		Body:    "## Approach\nEdit the file.\n\n## Acceptance criteria\n- tests pass",
	}
}

func TestNewPlanAcceptsValid(t *testing.T) {
	if _, err := NewPlan(validPlan()); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

func TestPlanRequiresSummaryAndBody(t *testing.T) {
	p := validPlan()
	p.Summary = ""
	p.Body = ""

	_, err := NewPlan(p)
	assertError(t, err, "plan.summary: empty\n"+
		"plan.body: empty")
}

func TestReviewRejectRequiresFeedback(t *testing.T) {
	_, err := NewReview(Review{
		TaskID:      "task-1",
		ExecutionID: "exec-1",
		Decision:    ReviewReject,
		Summary:     "does not compile",
	})
	assertError(t, err, "review.feedback: required when decision is reject")
}

func TestReviewAcceptAllowsOptionalFeedback(t *testing.T) {
	base := Review{
		TaskID:      "task-1",
		ExecutionID: "exec-1",
		Decision:    ReviewAccept,
		Summary:     "looks good",
	}

	if _, err := NewReview(base); err != nil {
		t.Fatalf("accept without feedback rejected: %v", err)
	}

	base.Feedback = "minor: naming nit in greet.ts, fine to ship"
	if _, err := NewReview(base); err != nil {
		t.Fatalf("accept with feedback rejected: %v", err)
	}
}

func TestReviewInvalidDecision(t *testing.T) {
	_, err := NewReview(Review{
		TaskID:      "task-1",
		ExecutionID: "exec-1",
		Decision:    "maybe",
		Summary:     "hmm",
	})
	assertError(t, err, `review.decision: invalid value "maybe"`)
}

func TestExecutionResultInvalidStatus(t *testing.T) {
	_, err := NewExecutionResult(ExecutionResult{
		TaskID: "task-1",
		PlanID: "plan-1",
		Status: "weird",
	})
	assertError(t, err, `execution_result.status: invalid value "weird"`)
}

func TestReviewJoinsMultipleErrorsInFieldOrder(t *testing.T) {
	_, err := NewReview(Review{})
	assertError(t, err, "review.task_id: empty\n"+
		"review.execution_id: empty\n"+
		"review.decision: empty\n"+
		"review.summary: empty")
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
