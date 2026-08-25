--
-- PostgreSQL database dump
--
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;
--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--
CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;
--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--
COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';
SET default_tablespace = '';
SET default_table_access_method = heap;
--
-- Name: idempotency_keys; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.idempotency_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    scope text NOT NULL,
    idempotency_key text NOT NULL,
    request_method text NOT NULL,
    request_path text NOT NULL,
    request_fingerprint bytea NOT NULL,
    status text NOT NULL,
    response_status integer,
    response_payload bytea,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT idempotency_keys_status_check CHECK ((status = ANY (ARRAY['claimed'::text, 'completed'::text])))
);
--
-- Name: TABLE idempotency_keys; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.idempotency_keys IS '冪等性キー';
--
-- Name: COLUMN idempotency_keys.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.id IS 'ID';
--
-- Name: COLUMN idempotency_keys.scope; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.scope IS 'スコープ（認証プリンシパルID）';
--
-- Name: COLUMN idempotency_keys.idempotency_key; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.idempotency_key IS '冪等性キー（クライアント供給）';
--
-- Name: COLUMN idempotency_keys.request_method; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.request_method IS 'リクエストメソッド';
--
-- Name: COLUMN idempotency_keys.request_path; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.request_path IS 'リクエストパス';
--
-- Name: COLUMN idempotency_keys.request_fingerprint; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.request_fingerprint IS 'リクエスト指紋（SHA-256）';
--
-- Name: COLUMN idempotency_keys.status; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.status IS '状態（claimed / completed）';
--
-- Name: COLUMN idempotency_keys.response_status; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.response_status IS 'レスポンスHTTPステータス（completedまでNULL）';
--
-- Name: COLUMN idempotency_keys.response_payload; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.response_payload IS 'レスポンスペイロード（結果DTOのJSONシリアライズ、completedまでNULL）';
--
-- Name: COLUMN idempotency_keys.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.created_at IS '作成日時';
--
-- Name: COLUMN idempotency_keys.completed_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.completed_at IS '完了日時';
--
-- Name: COLUMN idempotency_keys.expires_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.idempotency_keys.expires_at IS '有効期限（TTL）';
--
-- Name: outbox; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.outbox (
    id bigint NOT NULL,
    message_id uuid DEFAULT gen_random_uuid() NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    CONSTRAINT outbox_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'published'::text, 'dead'::text])))
);
--
-- Name: TABLE outbox; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.outbox IS 'トランザクショナル outbox（ドメインイベントの信頼 publish）';
--
-- Name: COLUMN outbox.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.id IS 'ID';
--
-- Name: COLUMN outbox.message_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.message_id IS 'dedup の安定キー（INSERT 時採番、Idempotency-Key へ伝搬）';
--
-- Name: COLUMN outbox.aggregate_type; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.aggregate_type IS '集約種別（観測・調査用。順序キーではない）';
--
-- Name: COLUMN outbox.aggregate_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.aggregate_id IS '集約ID（観測・調査用。順序キーではない）';
--
-- Name: COLUMN outbox.event_type; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.event_type IS 'イベント種別 + version';
--
-- Name: COLUMN outbox.payload; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.payload IS 'ペイロード（snapshot + version の収束可能な自己完結ペイロード）';
--
-- Name: COLUMN outbox.headers; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.headers IS 'publish 時に伝搬するヘッダ（traceparent 等）';
--
-- Name: COLUMN outbox.status; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.status IS '状態（pending / published / dead）';
--
-- Name: COLUMN outbox.attempts; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.attempts IS 'publish 試行回数';
--
-- Name: COLUMN outbox.last_error; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.last_error IS '直近の publish 失敗理由';
--
-- Name: COLUMN outbox.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.created_at IS '作成日時';
--
-- Name: COLUMN outbox.published_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.outbox.published_at IS 'publish 完了日時（published 遷移時刻）';
--
-- Name: outbox_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--
ALTER TABLE public.outbox ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);
--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);
--
-- Name: idempotency_keys idempotency_keys_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_id_primary PRIMARY KEY (id);
--
-- Name: idempotency_keys idempotency_keys_scope_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_scope_key_unique UNIQUE (scope, idempotency_key);
--
-- Name: outbox outbox_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.outbox
    ADD CONSTRAINT outbox_id_primary PRIMARY KEY (id);
--
-- Name: outbox outbox_message_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.outbox
    ADD CONSTRAINT outbox_message_id_unique UNIQUE (message_id);
--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);
--
-- Name: idempotency_keys_expires_at_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX idempotency_keys_expires_at_idx ON public.idempotency_keys USING btree (expires_at);
--
-- Name: outbox_dead_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX outbox_dead_idx ON public.outbox USING btree (id) WHERE (status = 'dead'::text);
--
-- Name: outbox_pending_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX outbox_pending_idx ON public.outbox USING btree (id) WHERE (status = 'pending'::text);
--
-- Name: outbox_published_gc_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX outbox_published_gc_idx ON public.outbox USING btree (published_at) WHERE (status = 'published'::text);
--
-- PostgreSQL database dump complete
--