package events

const (
	TopicArticleEvents     = "article.events"
	TopicInteractionEvents = "interaction.events"
	TopicRankingDLQ        = "ranking.dlq"

	EventArticlePublished  = "article.published"
	EventInteractionLike   = "interaction.like"
	EventInteractionUnlike = "interaction.unlike"
	EventInteractionRead   = "interaction.read"
)
