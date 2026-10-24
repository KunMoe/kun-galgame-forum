-- Replaces the 27 prefix-coded topic sections (g-/t-/o-) with 12 sections in
-- four display groups, and gives a topic exactly one section.
--
-- Why: the help sections cost moemoepoint, so users filed help requests under
-- chatting or other on purpose; three "other" sections and repeated names left
-- no section for many topics. The big category is no longer a topic attribute:
-- it becomes the group a section is shown under. topic.category keeps being
-- written with the group's legacy key because the App decodes it as required.
--
-- Existing rows:
--   * topic_section_relation: every topic keeps one relation, its first old
--     section by position mapped through the table below. The full old filing
--     is kept in topic_section_legacy_relation; the reclassification backfill
--     reads it (old g-other topics that carry resources go to the archived
--     resource-release section) and the down migration restores from it.
--   * topic.category is rewritten from the new section's group.
--   * topic_draft.sections / category are mapped the same way.
--   * user_purge_archive rows that captured a topic_section_relation point at
--     old section ids; they are rewritten so a later restore still finds them.
--   * The old 27 topic_section rows are deleted after nothing references them.
--
-- Additive for readers of topic_section_relation, but the API from this
-- release on writes and validates only the new keys: migrate with the deploy.

BEGIN;

CREATE TABLE IF NOT EXISTS topic_section_group (
  id              serial PRIMARY KEY,
  key             text NOT NULL UNIQUE,
  label           text NOT NULL,
  position        integer NOT NULL DEFAULT 0,
  legacy_category text NOT NULL CHECK (legacy_category IN ('galgame', 'technique', 'others')),
  created         timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated         timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE topic_section
  ADD COLUMN IF NOT EXISTS group_id          integer REFERENCES topic_section_group(id),
  ADD COLUMN IF NOT EXISTS label             text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS description       text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS icon              text NOT NULL DEFAULT 'lucide:hash',
  ADD COLUMN IF NOT EXISTS color             text NOT NULL DEFAULT 'default',
  ADD COLUMN IF NOT EXISTS position          integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS is_archived       boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS costs_moemoepoint boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS hidden_from_feeds boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS topic_section_legacy_relation (
  topic_id integer NOT NULL REFERENCES topic(id) ON DELETE CASCADE,
  name     text NOT NULL,
  position smallint NOT NULL,
  PRIMARY KEY (topic_id, name)
);

INSERT INTO topic_section_group (key, label, position, legacy_category) VALUES
  ('galgame',   'Galgame', 0, 'galgame'),
  ('resource',  '资源',    1, 'galgame'),
  ('technique', '技术',    2, 'technique'),
  ('community', '综合',    3, 'others')
ON CONFLICT (key) DO NOTHING;

INSERT INTO topic_section (name, updated, group_id, label, icon, color, position, is_archived, costs_moemoepoint, hidden_from_feeds, description)
SELECT v.name, CURRENT_TIMESTAMP, g.id, v.label, v.icon, v.color, v.position, v.is_archived, v.costs, v.hidden, v.description
FROM (VALUES
  ('galgame-discussion', 'galgame', 'Galgame 讨论', 'lucide:messages-square', 'primary', 0, false, false, false,
   '围绕 Galgame 本身展开交流的分区。你可以在这里聊正在玩或已经通关的作品、角色与剧情、声优与画师、会社动态，也可以发起推荐、比较、吐槽与感想分享，或者询问某部作品值不值得玩。篇幅不限，但请尽量说清你想讨论的作品与话题。系统性的长篇评测请发到「长文评价」，攻略与存档请发到「Galgame 攻略」，寻找资源请发到「寻求资源」，游戏运行时遇到的问题请发到「游戏运行」。'),
  ('galgame-walkthrough', 'galgame', 'Galgame 攻略', 'lucide:map', 'primary', 1, false, false, false,
   '发布与通关相关的实用内容：路线与选项攻略、全结局与全 CG 达成方法、存档分享、流程图、隐藏要素说明，以及与攻略配套的工具用法等。请在标题注明作品名，正文写清适用的版本（原版、汉化版、移植版）与语言，并标注涉及剧透的部分。转载他人的攻略请注明出处。如果只是想问某条线怎么走、某个选项怎么选，请写明你已经尝试过的步骤，单纯的提问请发到「Galgame 讨论」。'),
  ('review', 'galgame', '长文评价', 'lucide:book-open-text', 'primary', 2, false, false, false,
   '面向有分量的长篇内容：作品评测、通关感想、剧情与人物分析、系列或会社的回顾、题材与流派的考察，以及与 Galgame 相关的考据与长篇随笔。这里看重内容的完整度与个人见解，建议有清晰的结构，并在开头注明是否包含剧透。简短的一句话感想、推荐请求或闲聊请发到「Galgame 讨论」；与 Galgame 无关的文章请发到「ACGN」或「日常杂谈」。原创文章欢迎注明作者与首发地址。'),
  ('news', 'galgame', '新作资讯', 'lucide:newspaper', 'primary', 3, false, false, false,
   '发布 Galgame 相关的新消息：新作公布与发售日、体验版与官方情报、汉化与移植进度、会社动态、展会与活动、促销与限时免费，以及业界新闻等。请在标题写明作品或会社名称，正文附上消息来源链接，并尽量注明消息的日期；传闻与未经证实的消息请明确标注。对某条资讯的长篇看法可以发到「长文评价」，只是想聊聊某部新作请发到「Galgame 讨论」。本区不接受资源下载链接。'),
  ('resource-seeking', 'resource', '寻求资源', 'lucide:search', 'warning', 0, false, true, true,
   '寻找难以获得的 Galgame 及相关资源：游戏本体、补丁、汉化、原声、画集、设定集等。发布话题会消耗 10 萌萌点，话题不会出现在主页与全部话题列表中，你可以把最有帮助的回复采纳为最佳答案。请在标题写明作品名，正文写清你需要的版本、语言与平台，以及你已经找过哪些地方。本站已有资源的作品请先到作品页查找；分享资源请到作品页发布，不要发在话题中。'),
  ('resource-release', 'resource', '资源发布', 'lucide:archive', 'default', 1, true, false, false,
   '早期论坛里用户在话题中发布 Galgame 资源的归档分区。现在 Galgame 资源统一在作品页发布与管理，本分区已经归档，不再接受新的话题，只保留历史话题供大家查阅。如果你想分享资源，请前往对应作品的页面发布；如果找不到需要的资源，请发到「寻求资源」。历史话题中的链接可能已经失效，遇到问题欢迎在「站务相关」反馈，我们会尽量协助处理。'),
  ('tech-help', 'technique', '技术求助', 'lucide:life-buoy', 'warning', 0, false, true, true,
   '简短的技术提问放在这里，例如「这个游戏为什么打不开」「这个报错是怎么回事」，没有附带报错信息、截图或排查过程的问题都属于本区。发布话题会消耗 10 萌萌点，话题不会出现在主页与全部话题列表中。如果你写清了运行环境、完整的报错内容或截图，以及已经尝试过的方法，你的提问对其他人同样有参考价值，会被归入「游戏运行」或「开发技术」，不消耗萌萌点。'),
  ('game-running', 'technique', '游戏运行', 'lucide:monitor-play', 'success', 1, false, false, false,
   '讨论让 Galgame 正常运行的一切：安装与解压、转区与乱码、字体与分辨率、补丁与汉化的使用、模拟器与移植方案（如 Winlator、KiriKiri、ONScripter）、存档迁移，以及 Linux、macOS 与手机上的运行方法。这里欢迎有价值的提问：请写清系统与设备、游戏版本、完整的报错内容或截图，以及你已经尝试过的步骤。经验分享与教程同样欢迎。缺少这些信息的简短提问会被归入「技术求助」。'),
  ('development', 'technique', '开发技术', 'lucide:code', 'success', 2, false, false, false,
   '与开发和技术本身相关的交流：编程语言与框架、Web 与客户端开发、游戏引擎与脚本、汉化与逆向工程、机器学习与 AI 工具、Linux 与服务器、设计与多媒体软件、实用工具与网站分享等。欢迎教程、经验总结、项目展示与有价值的技术提问；提问请写清环境、你的目标、完整的报错与已经尝试的方法。只为让某个游戏跑起来的问题请发到「游戏运行」，缺少排查信息的简短提问会被归入「技术求助」。'),
  ('acgn', 'community', 'ACGN', 'lucide:sparkles', 'secondary', 0, false, false, false,
   'Galgame 以外的二次元内容：动画、漫画、轻小说、音乐与声优、其它类型的游戏、同人与周边等。可以发布新番讨论、作品推荐与感想、追番记录、歌曲分享、活动与展会见闻等。如果内容主要围绕某部 Galgame 或其改编作品，请优先发到「Galgame 讨论」；系统性的长篇评测可以发到「长文评价」。涉及剧透请在开头注明，本区同样不接受资源下载链接，分享资源请前往作品页。'),
  ('daily', 'community', '日常杂谈', 'lucide:coffee', 'secondary', 1, false, false, false,
   '与 ACGN 和技术都不直接相关的日常交流：生活记录、学习与工作、心情与随笔、兴趣爱好、有趣的见闻，以及暂时找不到合适分区的话题。可以轻松随意，但请保持友善，不要发布广告、引战或违反社区规则的内容。话题主要关于某部 Galgame 的，请发到「Galgame 讨论」；关于动画、漫画、轻小说或音乐的，请发到「ACGN」；关于本站的建议与反馈，请发到「站务相关」。'),
  ('site', 'community', '站务相关', 'lucide:megaphone', 'secondary', 2, false, false, false,
   '关于本站与 KUN Galgame 旗下站点的一切：功能建议、问题与 Bug 反馈、使用疑问、规则讨论、举报与申诉、补丁站（moyu.moe）相关的反馈，以及站点公告与活动。反馈问题时请写清页面地址、你的操作步骤、设备与浏览器，以及看到的提示或截图，这能帮助我们更快定位问题。对话题的分区有异议时，请直接在话题页申请管理员调整，不需要另外发帖。')
) AS v(name, group_key, label, icon, color, position, is_archived, costs, hidden, description)
JOIN topic_section_group g ON g.key = v.group_key
WHERE NOT EXISTS (SELECT 1 FROM topic_section s WHERE s.name = v.name);

CREATE TEMP TABLE section_regroup (old_name text PRIMARY KEY, new_name text NOT NULL) ON COMMIT DROP;
INSERT INTO section_regroup VALUES
  ('g-chatting', 'galgame-discussion'), ('g-other', 'galgame-discussion'),
  ('g-walkthrough', 'galgame-walkthrough'), ('g-article', 'review'),
  ('g-news', 'news'), ('g-releases', 'news'), ('g-seeking', 'resource-seeking'),
  ('t-help', 'tech-help'),
  ('t-android', 'game-running'), ('t-linux', 'game-running'), ('t-practical', 'game-running'),
  ('t-crack', 'development'), ('t-web', 'development'), ('t-languages', 'development'),
  ('t-ai', 'development'), ('t-algorithm', 'development'), ('t-adobe', 'development'),
  ('t-other', 'development'),
  ('o-anime', 'acgn'), ('o-comics', 'acgn'), ('o-music', 'acgn'), ('o-novel', 'acgn'),
  ('o-daily', 'daily'), ('o-essay', 'daily'), ('o-other', 'daily'),
  ('o-forum', 'site'), ('o-patch', 'site');

INSERT INTO topic_section_legacy_relation (topic_id, name, position)
SELECT r.topic_id, s.name, r.position
FROM topic_section_relation r
JOIN topic_section s ON s.id = r.topic_section_id
JOIN section_regroup m ON m.old_name = s.name
ON CONFLICT DO NOTHING;

CREATE TEMP TABLE topic_regroup ON COMMIT DROP AS
SELECT DISTINCT ON (r.topic_id) r.topic_id, ns.id AS section_id, g.legacy_category
FROM topic_section_relation r
JOIN topic_section os ON os.id = r.topic_section_id
JOIN section_regroup m ON m.old_name = os.name
JOIN topic_section ns ON ns.name = m.new_name
JOIN topic_section_group g ON g.id = ns.group_id
ORDER BY r.topic_id, r.position, r.topic_section_id;

DELETE FROM topic_section_relation r
USING topic_section s, section_regroup m
WHERE s.id = r.topic_section_id AND m.old_name = s.name;

INSERT INTO topic_section_relation (topic_id, topic_section_id, position, updated)
SELECT topic_id, section_id, 0, CURRENT_TIMESTAMP FROM topic_regroup
ON CONFLICT DO NOTHING;

UPDATE topic t SET category = x.legacy_category
FROM topic_regroup x
WHERE x.topic_id = t.id AND t.category IS DISTINCT FROM x.legacy_category;

UPDATE topic_draft d SET sections = to_jsonb(ARRAY[m.new_name])::text, category = g.legacy_category
FROM section_regroup m, topic_section ns, topic_section_group g
WHERE d.sections <> '' AND d.sections::jsonb ->> 0 = m.old_name
  AND ns.name = m.new_name AND g.id = ns.group_id;

UPDATE user_purge_archive a
SET row_data = jsonb_set(a.row_data, '{topic_section_id}', to_jsonb(ns.id)),
    row_pk = CASE WHEN a.row_pk ? 'topic_section_id' THEN jsonb_set(a.row_pk, '{topic_section_id}', to_jsonb(ns.id)) ELSE a.row_pk END
FROM topic_section os, section_regroup m, topic_section ns
WHERE a.table_name = 'topic_section_relation'
  AND os.id = (a.row_data ->> 'topic_section_id')::int
  AND m.old_name = os.name AND ns.name = m.new_name;

DELETE FROM topic_section s USING section_regroup m WHERE m.old_name = s.name;

CREATE UNIQUE INDEX IF NOT EXISTS topic_section_name_key ON topic_section (name);
CREATE UNIQUE INDEX IF NOT EXISTS topic_section_relation_one_per_topic ON topic_section_relation (topic_id);

SELECT user_purge_archive_attach();

COMMIT;
