package ai

import "fmt"

// RequestDecisionNotice is what the author reads when staff decide.
func RequestDecisionNotice(fromLabel, toLabel string, approved bool) string {
	if approved {
		return fmt.Sprintf("你申请把话题移到「%s」, 管理员已经同意并移动了话题", toLabel)
	}
	return fmt.Sprintf("你申请把话题移到「%s」, 管理员没有同意, 话题保留在「%s」", toLabel, fromLabel)
}

// RecheckReason is the request reason the sweep files for staff.
func RecheckReason(modelReason string) string {
	return "发布时模型不可用, 复核认为应在此分区: " + modelReason
}
