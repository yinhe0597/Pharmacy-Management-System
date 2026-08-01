// Package interaction 规则数据结构定义。
package interaction

// ExplicitPairRule 显式药品对交互规则（来自 drug_interactions 表）。
type ExplicitPairRule struct {
	DrugAID       int64
	DrugBID       int64
	Level         int
	Mechanism     string
	EvidenceLevel string
	Description   string
}

// IngredientRule 成分级交互规则（来自 ingredient_interactions 表）。
type IngredientRule struct {
	IngredientA   string
	IngredientB   string
	Level         int
	Mechanism     string
	EvidenceLevel string
	Description   string
}

// ClassRule 分类级交互规则（来自 class_interaction_rules 表）。
type ClassRule struct {
	ClassA        string
	ClassB        string
	Level         int
	Mechanism     string
	EvidenceLevel string
	Description   string
	IsActive      bool
}

// TagRule 标签级交互规则（来自 tag_interactions 表）。
type TagRule struct {
	TagA          string
	TagB          string
	Level         int
	Mechanism     string
	EvidenceLevel string
	Description   string
	IsActive      bool
}
