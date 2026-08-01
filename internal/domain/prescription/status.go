// Package prescription 定义处方状态机（纯领域，无依赖）。
package prescription

// Status 处方状态。
type Status string

// 处方状态取值。
const (
	StatusPendingReview    Status = "pending_review"    // 待审核
	StatusReviewedPassed   Status = "reviewed_passed"   // 审核通过
	StatusReviewedRejected Status = "reviewed_rejected" // 审核驳回
	StatusDispensing       Status = "dispensing"        // 调配中
	StatusDispensed        Status = "dispensed"         // 已发药
	StatusReturned         Status = "returned"          // 已退药
	StatusCancelled        Status = "cancelled"         // 已作废
)

// transitions 定义允许的状态流转及触发动作。
var transitions = map[Status]map[Status]string{
	StatusPendingReview: {
		StatusReviewedPassed:   "review_pass",
		StatusReviewedRejected: "review_reject",
		StatusCancelled:        "cancel",
	},
	StatusReviewedPassed: {
		StatusDispensing: "dispense",
		StatusCancelled:  "cancel",
	},
	StatusReviewedRejected: {
		StatusPendingReview: "re_submit",
		StatusCancelled:     "cancel",
	},
	StatusDispensing: {
		StatusDispensed: "confirm_dispense",
		StatusCancelled: "cancel",
	},
	StatusDispensed: {
		StatusReturned: "return",
	},
	StatusReturned:  {},
	StatusCancelled: {},
}

// String 返回状态字符串。
func (s Status) String() string { return string(s) }

// Valid 判断状态是否合法。
func (s Status) Valid() bool {
	_, ok := transitions[s]
	return ok
}

// CanTransitTo 判断从当前状态能否流转到 next，返回触发动作与是否允许。
func (s Status) CanTransitTo(next Status) (string, bool) {
	allowed, ok := transitions[s]
	if !ok {
		return "", false
	}
	action, ok := allowed[next]
	return action, ok
}

// IsTerminal 判断是否终态。
func (s Status) IsTerminal() bool {
	return s == StatusReturned || s == StatusCancelled
}
