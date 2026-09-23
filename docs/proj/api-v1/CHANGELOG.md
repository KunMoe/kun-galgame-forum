# API v1 changelog

## 2026-09-23 (G0 galgame id is the catalog work id)

Breaking for the one v1 galgame face.

- `GET /api/v1/galgames/{galgame_id}/moyu-patches` (`listGalgameMoyuPatches`) is now `GET /api/v1/works/{work_id}/moyu-patches` (`listWorkMoyuPatches`), tag `works`. Same response. The old path answers 404. No App build calls it.
- Every galgame id the site shows — the `/galgame/:id` page, every legacy `/api/*` field named `gid` or `galgame_id` — is now the catalog work id. 13,493 pages changed number in the renumber; merged-away numbers are not redirected. The legacy field names are unchanged until their faces move to v1.

## 2026-09-23 (D docs)

Breaking for every `/api/doc/**` face and `GET /api/admin/doc/article`; all sixteen are gone.

Offered:

- `GET /api/v1/docs`: cursor list of help-center docs (`limit` 1–100). `sort` is `position_asc` (default, the staff-set display order), `published_desc` or `views_desc`. Optional filters: `doc_category` and `is_pinned`.
- `GET /api/v1/docs/{doc_slug}`: one doc with `author` and `content` (the content-document node tree). Each read counts one view.
- `GET` / `PATCH` / `DELETE /api/v1/admin/docs/{doc_id}` and `POST /api/v1/admin/docs`: the staff view (`admin_doc`, with `content_markdown`), gated by `doc.edit` / `doc.create` / `doc.delete`.
- `PUT /api/v1/admin/doc-order`: `{doc_ids}` must name every doc exactly once.

Shape vs the retired faces: `doc_category` is a closed enum `galgame` / `notice` / `kun` / `other` and replaces `category_id` + `category{}`. `banner` is an `Image` or `null` and replaces `banner` / `banner_image_hash` / `banner_url`. `is_pinned` was `is_pin`, `view_count` was `view`, `published_at` was `published_time`, `edited_at` was `edited_time`. `path`, `status`, `sort_order`, `created`, `updated`, `tag_ids`, `content_html` and `toc` are gone; clients build `/doc/{slug}` themselves and take the table of contents from the `heading` nodes.

Removed without replacement: doc categories as their own resource (now the enum), doc tags (never used), and the draft/hidden `status` (never used; every doc is public).
## 2026-09-23 (WS website directory)

Offered:

- `GET /api/v1/websites` — cursor-paged directory, newest listing first; `include_nsfw` (default false), `website_category_id`, `website_tag_id`, `include_total`
- `GET /api/v1/websites/{website_host}` — detail by the page's host key; counts a view; `viewer` has `has_liked`, `has_favorited`, `can_edit`, `can_delete`
- `PUT` / `DELETE /api/v1/websites/{website_host}/like` and `/favorite` — idempotent slots returning `website_engagement`
- `GET /api/v1/website-categories`, `GET /api/v1/website-categories/{website_category_slug}`, `GET /api/v1/website-tags`, `GET /api/v1/website-tags/{website_tag_slug}`, `GET /api/v1/website-tag-groups`
- Staff (`website.*`, never a Bearer request): `POST /api/v1/admin/websites` and `GET` / `PATCH` / `DELETE /api/v1/admin/websites/{website_id}` (edit source `admin_website`); the same four for `website-categories`, `website-tags`, `website-tag-groups`
- New error code `WEBSITE_CATEGORY_NOT_EMPTY` (409, extension `website_count`)

Field names vs the retired faces: `host` (was `url` / the card's `domain`), `title` (was `name`), `urls` (was the `domain` array; every element is an http(s) URL), `founded` (was `create_time`), `is_nsfw` (was `age_limit`), `state` (was `status`, same values), `score` (was `price` and `level`), `icon: Image | null` plus `external_icon_url` (were `icon` / `icon_image_hash` / `icon_url`), `view_count` (was `view`), `website_category` / `website_tags` (were `category` / `tags`), `slug` / `label` on categories, tags and groups (were `name` / `label`), `website_tag_group_id` (was `group_id`), `is_multi_select` (was `multi_select`). The detail no longer carries `comment`; read the wall through `/api/v1/wall-comments`.

Retired: every `/api/website`, `/api/website-tag*`, `/api/website-category*` and `/api/website-tag-group` route (21).
## 2026-09-23 (P permissions)

Offered:

- `GET /api/v1/me/permissions` — the caller's effective permission keys in catalog order. Always empty for a Bearer request
- `GET` / `PATCH /api/v1/admin/role-permissions` — the role matrix; `PATCH {changes: [{role, overrides}]}` replaces the listed roles' overrides in one transaction, validated on the combined new state
- `GET` / `PUT /api/v1/admin/user-permissions/{user_id}` — one user's permission layer; `PUT {overrides}` replaces it
- `GET /api/v1/admin/permission-changes` — page-number collection of permission audit entries

Permission keys are an open vocabulary (`^[a-z][a-z_]*(\.[a-z][a-z_]*)+$`), so tolerate keys you do not know. Delegation refusals are `422 VALIDATION_FAILED` with `NOT_PERMITTED` at the refused position. Admin faces need the admin role; a Bearer request never has it.

Retired: `GET /api/perm/mine`, `GET /api/perm/bundles` (no replacement: it published the live role matrix to anonymous visitors), `GET /api/admin/role-permissions`, `PUT /api/admin/role-permissions/:role`, `GET`/`PUT /api/admin/user-permissions/:uid`, `GET /api/admin/permission-audit`.

## 2026-09-23 (TS reports and review inbox)

Offered:

- `GET /api/v1/report-reasons` — public; the reasons the trust service currently offers, `{key, display_name}`. Refreshed every five minutes; `503` when the trust service is down (there is no built-in fallback list any more)
- `POST /api/v1/reports` — `204`; `subject_kind` must be a kind this forum registers, `reason_key` one of the listed reasons, `subject_url` (optional) must start with `https://www.kungal.com/`. `Idempotency-Key` optional; repeating a report of the same content counts once
- `GET /api/v1/admin/review-items` — page-number collection (`page`, `limit` 1–100, optional `state`), needs `trust.review`; a Bearer request never has it
- `GET` / `PATCH /api/v1/admin/review-items/{review_item_id}` — the item with its reports; `PATCH {state: claimed | actioned | dismissed}` claims or decides it

Vocabulary: `state` `pending` / `claimed` / `actioned` / `dismissed`; `opened_by` is open (`reports`, `ai_text`, `ai_image`, `community_forward`, `mislabel`, `manual`, `ai_sample`, and `unknown` for an origin this forum does not know yet); `action` `none` / `hide` / `remove` / `warn_user` / `restrict` / `escalate_idp`. `subject_kind` and `reason_key` are open vocabularies.

Retired: `GET /api/report/reasons`, `POST /api/report/submit`, `GET /api/admin/trust/review-items`, `GET /api/admin/trust/review-items/:id`, `POST /api/admin/trust/review-items/:id/claim`, `POST /api/admin/trust/review-items/:id/decide`. `POST /api/trust/callback` stays: it is infra's HMAC callback, not an end-user face.

## 2026-09-23 (UP update logs and todo board)

Offered:

- `GET /api/v1/update-logs` — cursor-paged changelog, newest first; `GET /api/v1/update-logs/{update_log_id}`
- `POST` / `PATCH` / `DELETE /api/v1/update-logs…` — staff only (`update_log.create` / `.edit` / `.delete`); never available to a Bearer request
- `GET /api/v1/todos` — cursor-paged todo board, newest first; `state` filter and `include_total`; `GET /api/v1/todos/{todo_id}`
- `POST /api/v1/todos` — any signed-in user; `Idempotency-Key` required; trust-and-safety checked (`CONTENT_REJECTED`)
- `PATCH /api/v1/todos/{todo_id}` — either edit `project` / `text` (author only) or move `state`: claim, complete, discard, release, reopen. Every allowed move is mirrored by a `viewer.can_*` flag
- `DELETE /api/v1/todos/{todo_id}` — `update_log.delete`

Field names vs the retired faces: `change_type` (was `type`; tokens `perf` and `style`, were `pref` and `styles`), `release_version` (was `version`), `text` (was `content`, plain text, not Markdown), `project` (was the todo's `type`), `state` `pending` / `in_progress` / `done` / `discarded` (was integer `status` 0–3), `author` / `claimer` (were `user` / `claimed_user`), `completed_at` (was `completed_time`, now only set when `done`).

Retired: every `/api/update/**` route (`history` GET/POST/PUT/DELETE, `todo` GET/POST/PUT/DELETE, `todo/claim`, `todo/complete`, `todo/discard`).
## 2026-09-23 (U2 users)

Breaking for `GET /api/user/:id`, `GET /api/user/:id/floating`, and `GET`/`PUT /api/user/notification-preferences`.

Offered:

- `GET /api/v1/users/{user_id}` — public profile; unknown and banned/deleted users answer 404
- `GET /api/v1/users?ids=` — batch `UserRef` lookup (comma-separated ids, 1–100); requested ids that are unknown or not renderable sit in `missing`
- `GET` / `PUT /api/v1/me/notification-preferences` — `{ muted_types }` using v1 notification type tokens (`favorited`, `best_answer_chosen`, `reply_pinned`, `followed_thread_activity`, `lottery_drawn`, …; `chat` stays `chat`)

`UserProfile` counts sit under `counts` and are renamed (`topic` → `topic_count`, `reply_created` → `reply_count`, `galgame` → `published_galgame_count`, `galgame_toolset` → `toolset_count`, `upvote` → `received_upvote_count`, `daily_topic_count` → `topic_today_count`, …). `status` is gone.

Retired: `GET /api/user/:id`, `GET /api/user/:id/floating`, `GET`/`PUT /api/user/notification-preferences`.

## 2026-09-23 (U1 me)

Breaking for `/api/user/status` and the other U1 `/api/user/*` faces in this segment.

Offered:

- `GET /api/v1/me` — caller's cached moemoepoint, today's check-in gate, unread flag, creator flag, today's toolset upload bytes
- `POST /api/v1/me/check-ins` — Beijing-day check-in; reward is a deterministic function of that day (0–7)
- `GET /api/v1/me/moemoepoint-entries` — cursor ledger (`limit` 1–50); each entry has `source`: `this_site` / `account_center` / `other_site`
- `GET` / `PUT /api/v1/me/preferences` — cloud preference document; `If-Match` is the quoted version
- `PUT /api/v1/me/nsfw-display` — `hide` / `blur` / `show`
- `GET /api/v1/users?q=` — name search (`limit` 1–20), not paged
- `PATCH /api/v1/me/profile` — `name` and/or `bio`
- `PUT /api/v1/me/avatar` — multipart field `file`
- `GET /api/v1/me/creator-status` / `POST /api/v1/me/creator-applications`

`GET /api/v1/me` field names vs the retired status face: `moemoepoint` (was `moemoepoints`), `has_checked_in_today` (was `is_check_in`), `has_unread_messages` (was `has_new_message`; notifications and private messages, not system announcements), `toolset_upload_today_bytes` (was `daily_toolset_upload_bytes`).

Retired: `GET /api/user/status`. The rest of this segment's `/api/user/*` faces are gone with it (`check-in`, `moemoepoint/log`, `preferences`, `nsfw`, `search`, `bio`, `username`, `avatar`, `creator/status`, `creator/apply`). `GET`/`PUT /api/user/notification-preferences` stays until U2.

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

## 2026-09-23 (M · messages)

Twelve operations under `/api/v1/me/**` replace the eleven `/api/message/**` routes, which are deleted. The App never called the old routes, so nothing it shipped breaks.

Notifications:

- `GET /api/v1/me/notifications` (`listNotifications`) — cursor list, newest first. `is_muted` picks the partition (types the caller has not muted, or only the muted ones); `notification_type` narrows it. Rows whose actor is banned are left out.
- `GET /api/v1/me/notifications/summary` (`getNotificationSummary`) — `unread_count`, `muted_unread_count`, `latest`. The counts are SQL counts and include rows the list leaves out.
- `GET /api/v1/me/notifications/{notification_id}` (`getNotification`)
- `PUT /api/v1/me/notifications/read-marker` (`markNotificationsRead`) — `{up_to_id, is_muted}` marks one partition read up to an id. It never touches the other partition or rows newer than `up_to_id`.
- `DELETE /api/v1/me/notifications/{notification_id}` (`deleteNotification`) — `404` for a row that is not the caller's (the old route answered 200).
- `Notification.notification_type` is a closed vocabulary of 18 tokens. Five differ in meaning from their old stored names: `favorited`, `followed_thread_activity` (new comments in a thread you follow, not "someone followed you"), `best_answer_chosen`, `reply_pinned`, `resource_link_reported` (someone reported your link, not "your thing expired"). The other renames are snake_case only: `quiz_answered`, `edit_requested`, `edit_merged`, `edit_declined`, `lottery_won`, `lottery_drawn`, `lottery_code_expired`, `poll_closed`. The table is in `waves/m-message.md` §3.3.
- `path` is the in-site web path of the target. A structured target reference may be added later as an addition.

Private messages, addressed by the other participant's user id:

- `GET /api/v1/me/conversations` (`listConversations`), `GET /api/v1/me/conversations/{user_id}` (`getConversation`) — works before any message exists and creates nothing.
- `GET /api/v1/me/conversations/{user_id}/messages` (`listDirectMessages`), `GET …/messages/{message_id}` (`getDirectMessage`) — reading never marks anything read.
- `POST /api/v1/me/conversations/{user_id}/messages` (`sendDirectMessage`) — `Idempotency-Key` required. Returns `201` with the message.
- `PATCH /api/v1/me/conversations/{user_id}/messages/{message_id}` (`updateDirectMessage`) — `{state: "recalled"}`. A recalled message carries an empty `content` document.
- `PUT /api/v1/me/conversations/{user_id}/read-marker` (`markDirectMessagesRead`)

System announcements (`/api/message/admin`) are gone with no replacement. The table was empty and nothing wrote to it.
