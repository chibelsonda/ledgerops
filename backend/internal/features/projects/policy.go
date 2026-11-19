package projects

func canReviewPending(status, requestedByID, reviewerID string) bool {
	return status == "pending" && requestedByID != reviewerID
}

func revisedContractValue(originalValueMinor, approvedChangesMinor int64) int64 {
	return originalValueMinor + approvedChangesMinor
}

func canApplyChange(contractStatus string, originalValueMinor, approvedChangesMinor, allocatedMinor, changeMinor int64) bool {
	revised := revisedContractValue(originalValueMinor, approvedChangesMinor) + changeMinor
	return contractStatus == "approved" && revised > 0 && revised >= allocatedMinor
}

func canAllocateMilestone(contractStatus string, originalValueMinor, approvedChangesMinor, allocatedMinor, milestoneMinor int64) bool {
	return contractStatus == "approved" && allocatedMinor+milestoneMinor <= revisedContractValue(originalValueMinor, approvedChangesMinor)
}

func canInvoiceMilestone(milestoneStatus, contractStatus string) bool {
	return milestoneStatus == "planned" && contractStatus == "approved"
}
