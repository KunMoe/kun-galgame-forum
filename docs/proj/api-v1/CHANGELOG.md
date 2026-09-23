# API v1 changelog

## 2026-09-18

First `/api/v1` surface.

Offered:

- `GET /api/v1/problems` — error-code catalogue
- `GET /api/v1/problems/reasons` — field-level reason catalogue
- `GET /api/v1/topics` — cursor-paged topic list (`listTopics`)

Error codes in the registry (`pkg/problem`, `openapi/problems.json`):

- platform: `MALFORMED_BODY`, `INVALID_PARAMETER`, `UNKNOWN_ENUM_VALUE`, `LIMIT_TOO_LARGE`, `INVALID_CURSOR`, `UNKNOWN_SORT`, `MISSING_CREDENTIAL`, `INVALID_CREDENTIAL`, `SCOPE_REQUIRED`, `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `IDEMPOTENCY_KEY_REUSED`, `UNSUPPORTED_MEDIA_TYPE`, `VALIDATION_FAILED`, `INTERNAL_ERROR`, `SERVICE_UNAVAILABLE`
- kungal: `ACCOUNT_BANNED`, `IDEMPOTENCY_REQUEST_IN_PROGRESS`

## 2026-09-18 (W0b-1)

- The document no longer writes `"additionalProperties": true` on object schemas. It meant the same as leaving the keyword out, and unknown request-body fields are still accepted. Code generators rendered it as a catch-all map (openapi-typescript: `[key: string]: unknown`), which let a misspelled field compile. Regenerate client types.

## 2026-09-18 (W0b-3)

Both changes break clients generated from the earlier document. No deployment served that document.

- `TopicSummary.user` is now `author`. A field that refers to a person is named for the role that person plays in the resource; `user` is a forbidden name (gate G8).
- List bodies no longer declare `total`. None of the three collections accepts `include_total`, so the field could never appear. A collection that offers `include_total` returns a counted list that declares `total`, and gate F9 keeps the parameter and the field together.

## 2026-09-18 (W1 contract)

Additive.

- The document now defines the content node vocabulary: `ContentDocument`, the `BlockNode` and `InlineNode` unions (`oneOf`, discriminated on `object`) and their member schemas. No operation returns a document yet; the root extension `x-content-document` references it so it stays in the published document, and it goes away when the topic detail read returns `content`. Rules for rendering, including unknown node types, are in `docs/proj/api-v1/03-content-doc.md`.

## 2026-09-19 (W2)

Additive, apart from the removed extension.

- `GET /api/v1/topics/{topic_id}` (`getTopic`), `GET /api/v1/topics/{topic_id}/replies` (`listTopicReplies`), `GET /api/v1/replies/{reply_id}` (`getReply`) and `POST /api/v1/topics/{topic_id}/views` (`recordTopicView`).
- Reading a topic does not count a view. A client calls `recordTopicView` once when a reader actually opens the topic, and never when it prefetches or renders elsewhere.
- Replies are in floor order with a cursor; `from_floor` opens the list at a floor, and `sort=floor_desc` with `from_floor` one below reads back from it. The pinned reply and the best answer come with the topic and also appear at their floors.
- The root extension `x-content-document` is gone: `Topic.content` references `ContentDocument`.

## 2026-09-19 (moyu patches)

Additive.

- `GET /api/v1/galgames/{galgame_id}/moyu-patches` (`listGalgameMoyuPatches`): the pages www.moyu.moe holds for a galgame, each with its live resources. Public, never paged, and an answer may be up to 30 minutes old. It replaces the website's browser call to moyu's retired `/api/hikari`; the forum reads NextMoe's `/v2/moyu` face server-side.
- `types`, `languages`, `platforms` and `storage` are moyu's own vocabulary and are open (`x-vocabulary: moyu_patch_vocabulary`): show an unknown token as it is.
- `note_markdown` is moyu's Markdown source, not a content document. No download link, share code or password is carried; send a reader to `web_url`.
- `publisher` is a `UserRef` resolved from this forum's account service; the account is the same one on the forum.
- `SERVICE_UNAVAILABLE` when moyu, the catalog or the account service cannot be reached.

## 2026-09-19 (W3 / W4: topic writes and interactions)

Additive.

- Writes: `createTopic` (`POST /topics`), `updateTopic` (`PATCH /topics/{topic_id}`, including `state` for hiding and showing), `createReply` (`POST /topics/{topic_id}/replies`), `updateReply` (`PATCH /replies/{reply_id}`), `deleteReply` (`DELETE /replies/{reply_id}`). Both creates require `Idempotency-Key` and answer 201 with `Location` and the full resource.
- Edit context: `getTopicSource` and `getReplySource` return `content_markdown` and the editable fields to callers with `can_edit`. `content_markdown` is sent nowhere else.
- Interactions: `PUT` / `DELETE` on `/topics/{topic_id}/reactions/{reaction}`, `/topics/{topic_id}/favorite` and `/replies/{reply_id}/reactions/{reaction}` set and unset the caller's own state, answer 200 with an engagement snapshot, and are idempotent. like and dislike are the reaction tokens `like` and `dislike`.
- `upvoteTopic` (`POST /topics/{topic_id}/upvotes`) records a repeatable, charged upvote and requires `Idempotency-Key`; `listTopicUpvotes`, `listTopicReactions` and `listReplyReactions` are cursor lists, newest first.
- `setBestAnswer` / `clearBestAnswer` and `pinReply` / `unpinReply` on `/topics/{topic_id}/best-answer` and `/topics/{topic_id}/pinned-reply` answer 200 with the topic.
- `TopicViewer` gains `can_edit`, `can_hide`, `can_unhide`, `can_like`, `can_upvote`, `can_set_best_answer`, `can_pin_reply`; `ReplyViewer` gains `can_edit`, `can_delete`, `can_like`.
- New codes: `PERMISSION_REQUIRED` (moderation domain, as infra), `CONTENT_REJECTED`, `TOPIC_DAILY_LIMIT_REACHED` (with `limit`), `MOEMOEPOINT_INSUFFICIENT` (with `required`), `SELF_LIKE_FORBIDDEN`, `SELF_UPVOTE_FORBIDDEN`.
- `bumped_at` now also names upvotes, a new best answer, and edits of the title or body, which bump under the same 3-month rule.

## 2026-09-22 (legacy topic routes removed)

Breaking for anything still on `/api/topic/**`. Nothing was: the web moved with W1–W4 and the App binds `/api/v1` only (`docs/proj/app-direct-api.md`).

22 legacy routes are gone. Their v1 replacements, in the same order:

| removed | use instead |
|---|---|
| `POST /api/topic` | `POST /api/v1/topics` |
| `PUT /api/topic/:tid` | `PATCH /api/v1/topics/{topic_id}` |
| `GET /api/topic/:tid` | `GET /api/v1/topics/{topic_id}` |
| `PUT /api/topic/:tid/hide` | `PATCH /api/v1/topics/{topic_id}` with `state` |
| `PUT /api/topic/:tid/like`、`/dislike`、`/reaction` | `PUT` / `DELETE /api/v1/topics/{topic_id}/reactions/{reaction}` |
| `PUT /api/topic/:tid/favorite` | `PUT` / `DELETE /api/v1/topics/{topic_id}/favorite` |
| `PUT /api/topic/:tid/upvote` | `POST /api/v1/topics/{topic_id}/upvotes` |
| `GET /api/topic/:tid/upvotes` | `GET /api/v1/topics/{topic_id}/upvotes` |
| `GET /api/topic/:tid/reaction/history` | `GET /api/v1/topics/{topic_id}/reactions` |
| `PUT /api/topic/:tid/best-answer` | `PUT` / `DELETE /api/v1/topics/{topic_id}/best-answer` |
| `POST /api/topic/:tid/reply` | `POST /api/v1/topics/{topic_id}/replies` |
| `PUT /api/topic/:tid/reply` | `PATCH /api/v1/replies/{reply_id}` |
| `DELETE /api/topic/:tid/reply` | `DELETE /api/v1/replies/{reply_id}` |
| `GET /api/topic/:tid/reply` | `GET /api/v1/topics/{topic_id}/replies` |
| `GET /api/topic/:tid/reply/detail` | `GET /api/v1/replies/{reply_id}` |
| `PUT /api/topic/:tid/reply/like`、`/dislike`、`/reaction` | `PUT` / `DELETE /api/v1/replies/{reply_id}/reactions/{reaction}` |
| `PUT /api/topic/:tid/reply/pin` | `PUT` / `DELETE /api/v1/topics/{topic_id}/pinned-reply` |
| `GET /api/topic/:tid/reply/reaction/history` | `GET /api/v1/replies/{reply_id}/reactions` |

`internal/middleware/idempotency.go` went with them, and with it the envelope codes `237` / `238`. v1 idempotency is `internal/apiv1/idempotency.go`: `409 IDEMPOTENCY_REQUEST_IN_PROGRESS` and `409 IDEMPOTENCY_KEY_REUSED`.

A caller left on one of these now gets `401`, not `404`: the auth boundary in `router.go` is a `Use()` on `/api`, so every unmatched legacy path answers 401 to an anonymous request. That is how `/api/**` has always answered an unknown path — only `/api/v1/**` answers the proper `404` problem+json.

Still legacy, still mounted, each waiting on its own wave: `/topic/:tid/reply/locate`, `/topic/interactions/mine`, the four `/topic/draft*`, the four `/topic/:tid/comment*`, the six `/topic/:tid/poll*` and the eleven `/topic/:tid/lottery*`.

## 2026-09-22 (RC · comment walls)

Additive for the App; the web's legacy comment-wall routes are removed.

- Nine operations on one collection, `/api/v1/wall-comments`, serve the comment walls of six kinds of page: `listWallComments`, `createWallComment` (requires `Idempotency-Key`), `getWallComment`, `getWallCommentSource`, `updateWallComment`, `deleteWallComment`, `likeWallComment` / `unlikeWallComment` (`PUT` / `DELETE …/like`, both idempotent) and `flagWallComment` (`POST …/flags` → `204`).
- A wall is named by `subject_type` (`galgame`, `galgame_rating`, `galgame_resource`, `galgame_quiz`, `toolset`, `website`) and `subject_id`. The collection is flat because none of the six pages has a v1 read yet, and gate G17 refuses a write under a path id with no `GET`.
- `WallComment` is its own resource with its own id space; it is not a topic `Comment`. Its body is full Markdown as a `ContentDocument`. A deleted comment stays in lists as a tombstone (`state: deleted`, empty document) so its replies keep their parent. `addressee` is the person the comment is addressed to and can be `null`, which is why it is not called `in_reply_to_user`.
- The list sends no `total`. The page's own `comment_count` is the number to show.
- New error codes: `RATE_LIMITED` (platform, 429; the community service's new-account limit, no `Retry-After` because upstream sends none), `QUIZ_ANSWER_REQUIRED` (kungal, 403; the wall of a quiz that hides its game or has spoilers is open only to its author, those who answered, and staff with that wall's permissions) and `INVALID_STATE_TRANSITION` (me, 409; editing, liking or flagging a tombstone, or posting to a closed wall). It was named in 01 §2 but had never been registered.

Removed legacy routes (22):

| Legacy | v1 |
|---|---|
| `GET /api/{galgame-rating,website,toolset,galgame-resource,galgame-quiz}/:id/comments`, `GET /api/galgame/:gid/comments` | `GET /api/v1/wall-comments?subject_type=…&subject_id=…` |
| `POST` on the same six | `POST /api/v1/wall-comments` |
| `DELETE /api/{…five…}/:id/comments/:postId`, `DELETE /api/galgame/comments/:postId` | `DELETE /api/v1/wall-comments/{wall_comment_id}` |
| `PUT /api/galgame/comments/:postId` | `PATCH /api/v1/wall-comments/{wall_comment_id}` |
| `PUT /api/galgame/comments/:postId/like` (a toggle) | `PUT` / `DELETE /api/v1/wall-comments/{wall_comment_id}/like` |
| `POST /api/galgame/comments/:postId/flag` | `POST /api/v1/wall-comments/{wall_comment_id}/flags` |
| `GET /api/galgame/:gid/comments/locate` | not rebuilt; migration 135 rewrote the notification links that needed it |

Following a wall and its read receipts stay on the legacy `/api/community/wall/*` routes for now; `thread_id` may be sent as `0` and the server finds the thread.
