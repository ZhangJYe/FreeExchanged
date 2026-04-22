package constant

const (
	RankingHotKey = "ranking:hot"

	ArticleExchangeName      = "article.events"
	ArticlePublishQueueName  = "ranking_article_queue"
	ArticlePublishRoutingKey = "article.publish"

	InteractionExchangeName = "interaction.topic"
	InteractionQueueName    = "ranking_interaction_queue"
	InteractionRoutingKey   = "article.*"
)
