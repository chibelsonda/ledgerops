package projects

import "testing"

func TestContractPolicies(t *testing.T) {
	if canReviewPending("pending", "same-user", "same-user") {
		t.Fatal("requester must not review their own contract")
	}
	if !canReviewPending("pending", "requester", "reviewer") {
		t.Fatal("independent reviewer should be allowed")
	}
	if canApplyChange("approved", 100, 0, 90, -20) {
		t.Fatal("change order must not reduce value below allocated milestones")
	}
	if !canApplyChange("approved", 100, 0, 90, 20) {
		t.Fatal("valid change order was rejected")
	}
	if canAllocateMilestone("approved", 100, 0, 90, 20) {
		t.Fatal("milestone must not exceed revised contract value")
	}
}
