package models

// CategoryTypes 分类类型
type CategoryTypes string

const (
	// FeaturedCategory 特色文章
	FeaturedCategory CategoryTypes = "featured"

	// LatestCategory 最新文章
	LatestCategory CategoryTypes = "latest"

	// MostReadsCategory 最多阅读文章
	MostReadsCategory CategoryTypes = "most-reads"

	// OpinionsCategory 观点文章
	OpinionsCategory CategoryTypes = "opinions"

	// AnalysisCategory 分析文章
	AnalysisCategory CategoryTypes = "analysis"
)
