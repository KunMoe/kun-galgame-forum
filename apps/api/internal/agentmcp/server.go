package agentmcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"kun-galgame-api/internal/middleware"
	searchRepo "kun-galgame-api/internal/search/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/userclient"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Path = "/mcp"

type Profiles interface {
	FetchUserInfo(accessToken string) (*oauth.UserInfo, error)
}

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type Deps struct {
	Verifier middleware.AccessTokenVerifier
	Profiles Profiles
	Search   *searchRepo.SearchRepository
	Topics   *topicapiv1.Service
	Rows     *topicRepo.TopicRepository
	Taxonomy *topicRepo.TopicTaxonomyRepository
	Replies  *topicRepo.ReplyRepository
	Users    Users
	API      *ForumAPI
}

type server struct{ Deps }

func Handler(d Deps) http.Handler {
	s := &server{d}
	srv := mcp.NewServer(&mcp.Implementation{Name: "kungal", Title: "鲲 Galgame 论坛", Version: "1"}, &mcp.ServerOptions{
		Instructions: "这些工具查的是鲲 Galgame 论坛（kungal，本站）自己的内容：Galgame 和它们的下载资源（游戏本体、汉化补丁、合集等）、玩家评价、" +
			"话题（讨论、攻略、求助）、游戏引擎、Gal 工具和用户。只读，结果按提问用户在论坛里能看到的范围过滤。" +
			"问本站有没有某部作品、能不能下载、有没有中文、手机能不能玩、用什么引擎、玩家口碑、用什么工具时先用这些工具；" +
			"作品的剧情、角色、声优、画师、会社、系列、标签等资料，以及按题材找作品，用资料库工具（search_catalog、get_work、search_entities、get_entity、search_tags、browse_works），" +
			"再用 search_galgames / get_galgame 补上本站的资源情况。两边的作品 id 是同一个数。",
	})
	mcp.AddTool(srv, searchGalgamesTool(), s.searchGalgames)
	mcp.AddTool(srv, getGalgameTool(), s.getGalgame)
	mcp.AddTool(srv, searchResourcesTool(), s.searchResources)
	mcp.AddTool(srv, searchRatingsTool(), s.searchRatings)
	mcp.AddTool(srv, searchTopicsTool(), s.searchTopics)
	mcp.AddTool(srv, getTopicTool(), s.getTopic)
	mcp.AddTool(srv, searchEnginesTool(), s.searchEngines)
	mcp.AddTool(srv, getEngineTool(), s.getEngine)
	mcp.AddTool(srv, searchToolsetsTool(), s.searchToolsets)
	mcp.AddTool(srv, searchUsersTool(), s.searchUsers)
	mux := http.NewServeMux()
	mux.Handle(Path, mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	))
	return mux
}

type viewer struct {
	user        *middleware.UserInfo
	includeNSFW bool
}

var (
	errSignedOut   = errors.New("无法确认提问用户的身份：登录已失效或令牌无效。请提示用户重新登录鲲 Galgame 论坛后再试。")
	errUnavailable = errors.New("论坛暂时无法处理这个请求，请稍后再试。")
)

// The stance is folded the way the web folds it (shared/utils/contentStance.ts):
// anything short of an attested blur or show hides adult topics.
func (s *server) viewer(ctx context.Context, req *mcp.CallToolRequest) (viewer, error) {
	var header string
	if req.Extra != nil {
		header = req.Extra.Header.Get("Authorization")
	}
	scheme, token, _ := strings.Cut(strings.TrimSpace(header), " ")
	token = strings.TrimSpace(token)
	if !strings.EqualFold(scheme, "Bearer") || token == "" {
		return viewer{}, errSignedOut
	}
	claims, err := s.Verifier.Verify(ctx, token)
	if err != nil {
		if errors.Is(err, oauth.ErrKeysUnavailable) {
			slog.Error("agent MCP: JWKS unavailable", "error", err)
			return viewer{}, errUnavailable
		}
		return viewer{}, errSignedOut
	}
	v := viewer{user: middleware.BearerUser(claims)}
	info, err := s.Profiles.FetchUserInfo(token)
	if err != nil {
		slog.Warn("agent MCP: userinfo failed, hiding adult topics", "user_id", claims.ID, "error", err)
		return v, nil
	}
	v.includeNSFW = info.AdultConfirmed && (info.NSFWDisplay == "blur" || info.NSFWDisplay == "show")
	return v, nil
}
