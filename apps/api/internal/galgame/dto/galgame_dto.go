package dto

// GalgameIntro is one language's introduction. The language list is whatever
// catalog actually carries rather than a fixed set: the four product slots this
// replaces always shipped an empty 繁體中文, because catalog has never held a
// zh-Hant intro row, and the reader got a tab that could only say 暂无对应翻译.
type GalgameIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Machine bool   `json:"machine"`
}
