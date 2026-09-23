package dto

type TrustCallback struct {
	DispositionID int64  `json:"disposition_id"`
	SubjectKind   string `json:"subject_kind"`
	SubjectID     string `json:"subject_id"`
	Action        int16  `json:"action"`
	ReasonCode    string `json:"reason_code"`
}
