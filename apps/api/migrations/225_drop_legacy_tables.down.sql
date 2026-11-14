-- Recreates the tables and columns 225 dropped, empty, for rollback symmetry.
-- Their rows are not restored here: take them from the production dump made
-- on 2026-11-14 before 225 ran.
BEGIN;

CREATE TABLE IF NOT EXISTS public.chat_message_reaction (
    id integer NOT NULL,
    reaction text NOT NULL,
    chat_message_id integer NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.chat_message_reaction_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.chat_message_reaction_id_seq OWNED BY public.chat_message_reaction.id;

CREATE TABLE IF NOT EXISTS public.chat_room_admin (
    id integer NOT NULL,
    chat_room_id integer NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.chat_room_admin_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.chat_room_admin_id_seq OWNED BY public.chat_room_admin.id;

CREATE TABLE IF NOT EXISTS public.galgame_collection_item (
    id bigint NOT NULL,
    collection_id bigint NOT NULL,
    work_id integer CONSTRAINT galgame_collection_item_galgame_id_not_null NOT NULL,
    user_id integer NOT NULL,
    created timestamp with time zone DEFAULT now() NOT NULL,
    updated timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.galgame_collection_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.galgame_collection_item_id_seq OWNED BY public.galgame_collection_item.id;

CREATE TABLE IF NOT EXISTS public.galgame_collection_viewer (
    id bigint NOT NULL,
    collection_id bigint NOT NULL,
    user_id integer NOT NULL,
    created timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.galgame_collection_viewer_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.galgame_collection_viewer_id_seq OWNED BY public.galgame_collection_viewer.id;

CREATE TABLE IF NOT EXISTS public.galgame_favorite (
    id integer NOT NULL,
    work_id integer CONSTRAINT galgame_favorite_galgame_id_not_null NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.galgame_favorite_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.galgame_favorite_id_seq OWNED BY public.galgame_favorite.id;

CREATE TABLE IF NOT EXISTS public.galgame_resource_provider (
    id integer NOT NULL,
    resource_id integer NOT NULL,
    name character varying(255) NOT NULL,
    created timestamp(6) with time zone DEFAULT now() NOT NULL,
    updated timestamp(6) with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.galgame_resource_provider_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.galgame_resource_provider_id_seq OWNED BY public.galgame_resource_provider.id;

CREATE TABLE IF NOT EXISTS public.galgame_toolset_category (
    id integer NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    alias jsonb DEFAULT '[]'::jsonb,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.galgame_toolset_category_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.galgame_toolset_category_id_seq OWNED BY public.galgame_toolset_category.id;

CREATE TABLE IF NOT EXISTS public.galgame_toolset_category_relation (
    toolset_id integer NOT NULL,
    category_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE TABLE IF NOT EXISTS public.system_message (
    id integer NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL,
    content text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.system_message_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.system_message_id_seq OWNED BY public.system_message.id;

CREATE TABLE IF NOT EXISTS public.system_message_read_state (
    user_id integer NOT NULL,
    last_read_message_id bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.topic_dislike (
    id integer NOT NULL,
    topic_id integer NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.topic_dislike_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.topic_dislike_id_seq OWNED BY public.topic_dislike.id;

CREATE TABLE IF NOT EXISTS public.topic_like (
    id integer NOT NULL,
    topic_id integer NOT NULL,
    user_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.topic_like_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.topic_like_id_seq OWNED BY public.topic_like.id;

CREATE TABLE IF NOT EXISTS public.topic_reply_dislike (
    id integer NOT NULL,
    user_id integer NOT NULL,
    topic_reply_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.topic_reply_dislike_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.topic_reply_dislike_id_seq OWNED BY public.topic_reply_dislike.id;

CREATE TABLE IF NOT EXISTS public.topic_reply_like (
    id integer NOT NULL,
    user_id integer NOT NULL,
    topic_reply_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.topic_reply_like_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.topic_reply_like_id_seq OWNED BY public.topic_reply_like.id;

CREATE TABLE IF NOT EXISTS public.user_follow (
    id integer NOT NULL,
    follower_id integer NOT NULL,
    followed_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.user_follow_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.user_follow_id_seq OWNED BY public.user_follow.id;

CREATE TABLE IF NOT EXISTS public.user_friend (
    id integer NOT NULL,
    user_id integer NOT NULL,
    friend_id integer NOT NULL,
    created timestamp(3) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated timestamp(3) with time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.user_friend_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.user_friend_id_seq OWNED BY public.user_friend.id;

ALTER TABLE ONLY public.chat_message_reaction ALTER COLUMN id SET DEFAULT nextval('public.chat_message_reaction_id_seq'::regclass);

ALTER TABLE ONLY public.chat_room_admin ALTER COLUMN id SET DEFAULT nextval('public.chat_room_admin_id_seq'::regclass);

ALTER TABLE ONLY public.galgame_collection_item ALTER COLUMN id SET DEFAULT nextval('public.galgame_collection_item_id_seq'::regclass);

ALTER TABLE ONLY public.galgame_collection_viewer ALTER COLUMN id SET DEFAULT nextval('public.galgame_collection_viewer_id_seq'::regclass);

ALTER TABLE ONLY public.galgame_favorite ALTER COLUMN id SET DEFAULT nextval('public.galgame_favorite_id_seq'::regclass);

ALTER TABLE ONLY public.galgame_resource_provider ALTER COLUMN id SET DEFAULT nextval('public.galgame_resource_provider_id_seq'::regclass);

ALTER TABLE ONLY public.galgame_toolset_category ALTER COLUMN id SET DEFAULT nextval('public.galgame_toolset_category_id_seq'::regclass);

ALTER TABLE ONLY public.system_message ALTER COLUMN id SET DEFAULT nextval('public.system_message_id_seq'::regclass);

ALTER TABLE ONLY public.topic_dislike ALTER COLUMN id SET DEFAULT nextval('public.topic_dislike_id_seq'::regclass);

ALTER TABLE ONLY public.topic_like ALTER COLUMN id SET DEFAULT nextval('public.topic_like_id_seq'::regclass);

ALTER TABLE ONLY public.topic_reply_dislike ALTER COLUMN id SET DEFAULT nextval('public.topic_reply_dislike_id_seq'::regclass);

ALTER TABLE ONLY public.topic_reply_like ALTER COLUMN id SET DEFAULT nextval('public.topic_reply_like_id_seq'::regclass);

ALTER TABLE ONLY public.user_follow ALTER COLUMN id SET DEFAULT nextval('public.user_follow_id_seq'::regclass);

ALTER TABLE ONLY public.user_friend ALTER COLUMN id SET DEFAULT nextval('public.user_friend_id_seq'::regclass);

ALTER TABLE ONLY public.chat_message_reaction
    ADD CONSTRAINT chat_message_reaction_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.chat_room_admin
    ADD CONSTRAINT chat_room_admin_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_collection_item
    ADD CONSTRAINT galgame_collection_item_collection_id_galgame_id_key UNIQUE (collection_id, work_id);

ALTER TABLE ONLY public.galgame_collection_item
    ADD CONSTRAINT galgame_collection_item_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_collection_viewer
    ADD CONSTRAINT galgame_collection_viewer_collection_id_user_id_key UNIQUE (collection_id, user_id);

ALTER TABLE ONLY public.galgame_collection_viewer
    ADD CONSTRAINT galgame_collection_viewer_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_favorite
    ADD CONSTRAINT galgame_favorite_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_resource_provider
    ADD CONSTRAINT galgame_resource_provider_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_resource_provider
    ADD CONSTRAINT galgame_resource_provider_resource_id_name_key UNIQUE (resource_id, name);

ALTER TABLE ONLY public.galgame_toolset_category
    ADD CONSTRAINT galgame_toolset_category_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.galgame_toolset_category_relation
    ADD CONSTRAINT galgame_toolset_category_relation_pkey PRIMARY KEY (toolset_id, category_id);

ALTER TABLE ONLY public.system_message
    ADD CONSTRAINT system_message_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.system_message_read_state
    ADD CONSTRAINT system_message_read_state_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY public.topic_dislike
    ADD CONSTRAINT topic_dislike_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.topic_like
    ADD CONSTRAINT topic_like_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.topic_reply_dislike
    ADD CONSTRAINT topic_reply_dislike_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.topic_reply_like
    ADD CONSTRAINT topic_reply_like_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.user_follow
    ADD CONSTRAINT user_follow_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.user_friend
    ADD CONSTRAINT user_friend_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX IF NOT EXISTS chat_message_reaction_chat_message_id_user_id_reaction_key ON public.chat_message_reaction USING btree (chat_message_id, user_id, reaction);

CREATE UNIQUE INDEX IF NOT EXISTS chat_room_admin_chat_room_id_user_id_key ON public.chat_room_admin USING btree (chat_room_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS galgame_favorite_galgame_id_user_id_key ON public.galgame_favorite USING btree (work_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS galgame_toolset_category_name_key ON public.galgame_toolset_category USING btree (name);

CREATE INDEX IF NOT EXISTS idx_gci_galgame ON public.galgame_collection_item USING btree (work_id);

CREATE INDEX IF NOT EXISTS idx_gci_user_galgame ON public.galgame_collection_item USING btree (user_id, work_id);

CREATE INDEX IF NOT EXISTS idx_grp_resource_id ON public.galgame_resource_provider USING btree (resource_id);

CREATE UNIQUE INDEX IF NOT EXISTS topic_dislike_topic_id_user_id_key ON public.topic_dislike USING btree (topic_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS topic_like_topic_id_user_id_key ON public.topic_like USING btree (topic_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS topic_reply_dislike_user_id_topic_reply_id_key ON public.topic_reply_dislike USING btree (user_id, topic_reply_id);

CREATE UNIQUE INDEX IF NOT EXISTS topic_reply_like_user_id_topic_reply_id_key ON public.topic_reply_like USING btree (user_id, topic_reply_id);

CREATE UNIQUE INDEX IF NOT EXISTS user_follow_follower_id_followed_id_key ON public.user_follow USING btree (follower_id, followed_id);

CREATE UNIQUE INDEX IF NOT EXISTS user_friend_user_id_friend_id_key ON public.user_friend USING btree (user_id, friend_id);

ALTER TABLE ONLY public.chat_message_reaction
    ADD CONSTRAINT chat_message_reaction_chat_message_id_fkey FOREIGN KEY (chat_message_id) REFERENCES public.chat_message(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.chat_room_admin
    ADD CONSTRAINT chat_room_admin_chat_room_id_fkey FOREIGN KEY (chat_room_id) REFERENCES public.chat_room(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_collection_item
    ADD CONSTRAINT galgame_collection_item_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.galgame_collection(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_collection_viewer
    ADD CONSTRAINT galgame_collection_viewer_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.galgame_collection(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_favorite
    ADD CONSTRAINT galgame_favorite_galgame_id_fkey FOREIGN KEY (work_id) REFERENCES public.galgame(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_resource_provider
    ADD CONSTRAINT galgame_resource_provider_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.galgame_resource(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_toolset_category_relation
    ADD CONSTRAINT galgame_toolset_category_relation_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.galgame_toolset_category(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.galgame_toolset_category_relation
    ADD CONSTRAINT galgame_toolset_category_relation_toolset_id_fkey FOREIGN KEY (toolset_id) REFERENCES public.galgame_toolset(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.topic_dislike
    ADD CONSTRAINT topic_dislike_topic_id_fkey FOREIGN KEY (topic_id) REFERENCES public.topic(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.topic_like
    ADD CONSTRAINT topic_like_topic_id_fkey FOREIGN KEY (topic_id) REFERENCES public.topic(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.topic_reply_dislike
    ADD CONSTRAINT topic_reply_dislike_topic_reply_id_fkey FOREIGN KEY (topic_reply_id) REFERENCES public.topic_reply(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.topic_reply_like
    ADD CONSTRAINT topic_reply_like_topic_reply_id_fkey FOREIGN KEY (topic_reply_id) REFERENCES public.topic_reply(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE chat_message ADD COLUMN IF NOT EXISTS edit_time timestamp(3) with time zone;
ALTER TABLE galgame_activity ADD COLUMN IF NOT EXISTS wiki_pr_id bigint;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_galgame_activity_wiki_pr_id ON public.galgame_activity USING btree (wiki_pr_id);
ALTER TABLE kungal_user_state ADD COLUMN IF NOT EXISTS following_activity_seen_at timestamp with time zone;

SELECT user_purge_archive_attach();

COMMIT;
