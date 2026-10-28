package app

import (
	"net/http"

	"kun-galgame-api/internal/agentmcp"
	searchRepo "kun-galgame-api/internal/search/repository"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/user/oauth"
)

func (a *App) newAgentMCP() http.Handler {
	// The agent forwards the token the forum's web session holds, which the OP
	// issued to the web client rather than to any App client.
	clients := append([]string{a.Config.OAuth.ClientID}, a.Config.Bearer.ClientIDs...)
	return agentmcp.Handler(agentmcp.Deps{
		Verifier: oauth.NewAccessTokenVerifier(oauth.NewJWKS(a.Config.Bearer.JWKSURL), a.Config.Bearer.Issuer, clients),
		Profiles: a.OAuthClient,
		Search:   searchRepo.NewSearchRepository(a.DB),
		Topics:   a.newTopicV1(),
		Rows:     topicRepo.NewTopicRepository(a.DB),
		Taxonomy: topicRepo.NewTopicTaxonomyRepository(a.DB),
		Replies:  topicRepo.NewReplyRepository(a.DB),
		Users:    a.UserClient,
	})
}
