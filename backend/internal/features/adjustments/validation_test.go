package adjustments

import "testing"

func TestValidateCreateRequiresDerivedReversalAmount(t *testing.T) {
	input := CreateInput{
		Kind:        KindPaymentReversal,
		PaymentID:   "31111111-1111-4111-8111-111111111111",
		AmountMinor: 100,
		Reason:      "Duplicate bank receipt",
	}
	if message := validateCreate(input); message == "" {
		t.Fatal("expected client-provided reversal amount to be rejected")
	}
}

func TestValidateCreateAcceptsCreditWithJustification(t *testing.T) {
	input := CreateInput{Kind: KindCreditNote, AmountMinor: 10_000, Reason: "Approved service concession"}
	if message := validateCreate(input); message != "" {
		t.Fatalf("unexpected validation message: %s", message)
	}
}

func TestValidateDecisionRequiresReviewNote(t *testing.T) {
	if message := validateDecision(DecisionInput{Decision: "approved", Note: "no"}); message == "" {
		t.Fatal("expected short review note to be rejected")
	}
}
