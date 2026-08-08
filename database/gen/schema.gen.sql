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
-- Name: prefectures; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.prefectures (
    id uuid NOT NULL,
    name character varying(100) NOT NULL,
    code smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE prefectures; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.prefectures IS '都道府県';
--
-- Name: COLUMN prefectures.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.prefectures.id IS 'ID';
--
-- Name: COLUMN prefectures.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.prefectures.name IS '都道府県名';
--
-- Name: COLUMN prefectures.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.prefectures.code IS '都道府県コード';
--
-- Name: COLUMN prefectures.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.prefectures.created_at IS '作成日時';
--
-- Name: COLUMN prefectures.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.prefectures.updated_at IS '更新日時';
--
-- Name: product_categories; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.product_categories (
    id uuid NOT NULL,
    name character varying(100) NOT NULL,
    code smallint NOT NULL,
    sort_key smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE product_categories; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.product_categories IS '商品カテゴリ';
--
-- Name: COLUMN product_categories.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.id IS 'ID';
--
-- Name: COLUMN product_categories.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.name IS '名称';
--
-- Name: COLUMN product_categories.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.code IS 'コード';
--
-- Name: COLUMN product_categories.sort_key; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.sort_key IS '順序';
--
-- Name: COLUMN product_categories.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.created_at IS '作成日時';
--
-- Name: COLUMN product_categories.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_categories.updated_at IS '更新日時';
--
-- Name: product_statuses; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.product_statuses (
    id uuid NOT NULL,
    name character varying(100) NOT NULL,
    code smallint NOT NULL,
    sort_key smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE product_statuses; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.product_statuses IS '商品ステータス';
--
-- Name: COLUMN product_statuses.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.id IS 'ID';
--
-- Name: COLUMN product_statuses.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.name IS '名称';
--
-- Name: COLUMN product_statuses.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.code IS 'コード';
--
-- Name: COLUMN product_statuses.sort_key; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.sort_key IS '順序';
--
-- Name: COLUMN product_statuses.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.created_at IS '作成日時';
--
-- Name: COLUMN product_statuses.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.product_statuses.updated_at IS '更新日時';
--
-- Name: products; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.products (
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    price numeric NOT NULL,
    quantity integer NOT NULL,
    stock_warning_threshold integer,
    status_id uuid NOT NULL,
    category_id uuid NOT NULL,
    published_at timestamp with time zone,
    image_path text,
    lock_version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE products; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.products IS '商品';
--
-- Name: COLUMN products.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.id IS 'ID';
--
-- Name: COLUMN products.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.name IS '名称';
--
-- Name: COLUMN products.description; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.description IS '説明';
--
-- Name: COLUMN products.price; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.price IS '価格';
--
-- Name: COLUMN products.quantity; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.quantity IS '在庫数';
--
-- Name: COLUMN products.stock_warning_threshold; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.stock_warning_threshold IS '在庫警告閾値';
--
-- Name: COLUMN products.status_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.status_id IS '商品ステータスID';
--
-- Name: COLUMN products.category_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.category_id IS '商品カテゴリID';
--
-- Name: COLUMN products.published_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.published_at IS '公開日時';
--
-- Name: COLUMN products.image_path; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.image_path IS '画像パス';
--
-- Name: COLUMN products.lock_version; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.lock_version IS '楽観ロックバージョン';
--
-- Name: COLUMN products.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.created_at IS '作成日時';
--
-- Name: COLUMN products.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.products.updated_at IS '更新日時';
--
-- Name: purchase_details; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.purchase_details (
    id uuid NOT NULL,
    purchase_id uuid NOT NULL,
    product_id uuid NOT NULL,
    quantity integer NOT NULL,
    unit_price numeric NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE purchase_details; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.purchase_details IS '購入詳細';
--
-- Name: COLUMN purchase_details.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.id IS 'ID';
--
-- Name: COLUMN purchase_details.purchase_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.purchase_id IS '購入ID';
--
-- Name: COLUMN purchase_details.product_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.product_id IS '商品ID';
--
-- Name: COLUMN purchase_details.quantity; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.quantity IS '数量';
--
-- Name: COLUMN purchase_details.unit_price; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.unit_price IS '単価';
--
-- Name: COLUMN purchase_details.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.created_at IS '作成日時';
--
-- Name: COLUMN purchase_details.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_details.updated_at IS '更新日時';
--
-- Name: purchase_statuses; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.purchase_statuses (
    id uuid NOT NULL,
    name character varying(100) NOT NULL,
    code smallint NOT NULL,
    sort_key smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE purchase_statuses; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.purchase_statuses IS '購入ステータス';
--
-- Name: COLUMN purchase_statuses.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.id IS 'ID';
--
-- Name: COLUMN purchase_statuses.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.name IS '名称';
--
-- Name: COLUMN purchase_statuses.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.code IS 'コード';
--
-- Name: COLUMN purchase_statuses.sort_key; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.sort_key IS '順序';
--
-- Name: COLUMN purchase_statuses.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.created_at IS '作成日時';
--
-- Name: COLUMN purchase_statuses.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchase_statuses.updated_at IS '更新日時';
--
-- Name: purchases; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.purchases (
    id uuid NOT NULL,
    code character varying(50) NOT NULL,
    user_id uuid NOT NULL,
    status_id uuid NOT NULL,
    subtotal_amount bigint NOT NULL,
    tax_amount bigint NOT NULL,
    shipping_fee bigint NOT NULL,
    total_amount bigint NOT NULL,
    ordered_at timestamp with time zone DEFAULT now() NOT NULL,
    paid_at timestamp with time zone,
    canceled_at timestamp with time zone,
    shipped_at timestamp with time zone,
    delivered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE purchases; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.purchases IS '購入';
--
-- Name: COLUMN purchases.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.id IS 'ID';
--
-- Name: COLUMN purchases.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.code IS 'コード';
--
-- Name: COLUMN purchases.user_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.user_id IS 'ユーザID';
--
-- Name: COLUMN purchases.status_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.status_id IS '購入ステータスID';
--
-- Name: COLUMN purchases.subtotal_amount; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.subtotal_amount IS '小計金額';
--
-- Name: COLUMN purchases.tax_amount; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.tax_amount IS '税金額';
--
-- Name: COLUMN purchases.shipping_fee; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.shipping_fee IS '配送料';
--
-- Name: COLUMN purchases.total_amount; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.total_amount IS '合計金額';
--
-- Name: COLUMN purchases.ordered_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.ordered_at IS '注文日時';
--
-- Name: COLUMN purchases.paid_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.paid_at IS '支払日時';
--
-- Name: COLUMN purchases.canceled_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.canceled_at IS 'キャンセル日時';
--
-- Name: COLUMN purchases.shipped_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.shipped_at IS '発送日時';
--
-- Name: COLUMN purchases.delivered_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.delivered_at IS '配達日時';
--
-- Name: COLUMN purchases.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.created_at IS '作成日時';
--
-- Name: COLUMN purchases.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.purchases.updated_at IS '更新日時';
--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.roles (
    id uuid NOT NULL,
    name character varying(100) NOT NULL,
    code smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE roles; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.roles IS 'ロール';
--
-- Name: COLUMN roles.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.roles.id IS 'ID';
--
-- Name: COLUMN roles.name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.roles.name IS '名称';
--
-- Name: COLUMN roles.code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.roles.code IS 'コード';
--
-- Name: COLUMN roles.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.roles.created_at IS '作成日時';
--
-- Name: COLUMN roles.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.roles.updated_at IS '更新日時';
--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);
--
-- Name: user_identities; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.user_identities (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    issuer character varying(255) NOT NULL,
    subject character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE user_identities; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.user_identities IS '外部ID連携';
--
-- Name: COLUMN user_identities.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.id IS 'ID';
--
-- Name: COLUMN user_identities.user_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.user_id IS 'ユーザID';
--
-- Name: COLUMN user_identities.issuer; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.issuer IS 'トークン発行者（IdP issuer）';
--
-- Name: COLUMN user_identities.subject; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.subject IS '認証主体（token の sub）';
--
-- Name: COLUMN user_identities.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.created_at IS '作成日時';
--
-- Name: COLUMN user_identities.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_identities.updated_at IS '更新日時';
--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.user_roles (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: TABLE user_roles; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.user_roles IS 'ユーザロール';
--
-- Name: COLUMN user_roles.user_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_roles.user_id IS 'ユーザID';
--
-- Name: COLUMN user_roles.role_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_roles.role_id IS 'ロールID';
--
-- Name: COLUMN user_roles.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_roles.created_at IS '作成日時';
--
-- Name: COLUMN user_roles.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.user_roles.updated_at IS '更新日時';
--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.users (
    id uuid NOT NULL,
    first_name character varying(100) NOT NULL,
    last_name character varying(100) NOT NULL,
    email character varying(100) NOT NULL,
    phone character varying(20) NOT NULL,
    prefecture_id uuid NOT NULL,
    city character varying(100) NOT NULL,
    street character varying(255) NOT NULL,
    building character varying(255),
    postal_code character varying(8) NOT NULL,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    search_text text GENERATED ALWAYS AS ((((((((((((((((COALESCE(first_name, ''::character varying))::text || ' '::text) || (COALESCE(last_name, ''::character varying))::text) || ' '::text) || (COALESCE(email, ''::character varying))::text) || ' '::text) || (COALESCE(phone, ''::character varying))::text) || ' '::text) || (COALESCE(city, ''::character varying))::text) || ' '::text) || (COALESCE(street, ''::character varying))::text) || ' '::text) || (COALESCE(building, ''::character varying))::text) || ' '::text) || (COALESCE(postal_code, ''::character varying))::text)) STORED
);
--
-- Name: TABLE users; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON TABLE public.users IS 'ユーザ';
--
-- Name: COLUMN users.id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.id IS 'ID';
--
-- Name: COLUMN users.first_name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.first_name IS '名前';
--
-- Name: COLUMN users.last_name; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.last_name IS '苗字';
--
-- Name: COLUMN users.email; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.email IS 'メールアドレス';
--
-- Name: COLUMN users.phone; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.phone IS '電話番号';
--
-- Name: COLUMN users.prefecture_id; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.prefecture_id IS '都道府県ID';
--
-- Name: COLUMN users.city; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.city IS '市区町村';
--
-- Name: COLUMN users.street; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.street IS '番地';
--
-- Name: COLUMN users.building; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.building IS '建物名';
--
-- Name: COLUMN users.postal_code; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.postal_code IS '郵便番号';
--
-- Name: COLUMN users.deleted_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.deleted_at IS '削除日時';
--
-- Name: COLUMN users.created_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.created_at IS '作成日時';
--
-- Name: COLUMN users.updated_at; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.updated_at IS '更新日時';
--
-- Name: COLUMN users.search_text; Type: COMMENT; Schema: public; Owner: -
--
COMMENT ON COLUMN public.users.search_text IS '全文検索用テキスト';
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
-- Name: prefectures prefectures_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.prefectures
    ADD CONSTRAINT prefectures_code_unique UNIQUE (code);
--
-- Name: prefectures prefectures_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.prefectures
    ADD CONSTRAINT prefectures_id_primary PRIMARY KEY (id);
--
-- Name: prefectures prefectures_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.prefectures
    ADD CONSTRAINT prefectures_name_unique UNIQUE (name);
--
-- Name: product_categories product_categories_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_code_unique UNIQUE (code);
--
-- Name: product_categories product_categories_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_id_primary PRIMARY KEY (id);
--
-- Name: product_categories product_categories_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_name_unique UNIQUE (name);
--
-- Name: product_categories product_categories_sort_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_sort_key_unique UNIQUE (sort_key);
--
-- Name: product_statuses product_statuses_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_statuses
    ADD CONSTRAINT product_statuses_code_unique UNIQUE (code);
--
-- Name: product_statuses product_statuses_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_statuses
    ADD CONSTRAINT product_statuses_id_primary PRIMARY KEY (id);
--
-- Name: product_statuses product_statuses_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_statuses
    ADD CONSTRAINT product_statuses_name_unique UNIQUE (name);
--
-- Name: product_statuses product_statuses_sort_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.product_statuses
    ADD CONSTRAINT product_statuses_sort_key_unique UNIQUE (sort_key);
--
-- Name: products products_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_id_primary PRIMARY KEY (id);
--
-- Name: purchase_details purchase_details_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_details
    ADD CONSTRAINT purchase_details_id_primary PRIMARY KEY (id);
--
-- Name: purchase_statuses purchase_statuses_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_statuses
    ADD CONSTRAINT purchase_statuses_code_unique UNIQUE (code);
--
-- Name: purchase_statuses purchase_statuses_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_statuses
    ADD CONSTRAINT purchase_statuses_id_primary PRIMARY KEY (id);
--
-- Name: purchase_statuses purchase_statuses_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_statuses
    ADD CONSTRAINT purchase_statuses_name_unique UNIQUE (name);
--
-- Name: purchase_statuses purchase_statuses_sort_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_statuses
    ADD CONSTRAINT purchase_statuses_sort_key_unique UNIQUE (sort_key);
--
-- Name: purchases purchases_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchases
    ADD CONSTRAINT purchases_code_unique UNIQUE (code);
--
-- Name: purchases purchases_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchases
    ADD CONSTRAINT purchases_id_primary PRIMARY KEY (id);
--
-- Name: roles roles_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_code_unique UNIQUE (code);
--
-- Name: roles roles_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_id_primary PRIMARY KEY (id);
--
-- Name: roles roles_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_name_unique UNIQUE (name);
--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);
--
-- Name: user_identities user_identities_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_id_primary PRIMARY KEY (id);
--
-- Name: user_identities user_identities_issuer_subject_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_issuer_subject_unique UNIQUE (issuer, subject);
--
-- Name: user_identities user_identities_user_id_issuer_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_issuer_unique UNIQUE (user_id, issuer);
--
-- Name: user_roles user_roles_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_primary PRIMARY KEY (user_id, role_id);
--
-- Name: users users_email_unique; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_unique UNIQUE (email);
--
-- Name: users users_id_primary; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_id_primary PRIMARY KEY (id);
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
-- Name: products_low_stock_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX products_low_stock_idx ON public.products USING btree (quantity, id) WHERE (stock_warning_threshold IS NOT NULL);
--
-- Name: purchases_user_id_ordered_at_id_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX purchases_user_id_ordered_at_id_idx ON public.purchases USING btree (user_id, ordered_at DESC, id DESC);
--
-- Name: users_search_text_trgm_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX users_search_text_trgm_idx ON public.users USING gin (search_text public.gin_trgm_ops);
--
-- Name: products products_category_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_category_id_foreign FOREIGN KEY (category_id) REFERENCES public.product_categories(id);
--
-- Name: products products_status_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_status_id_foreign FOREIGN KEY (status_id) REFERENCES public.product_statuses(id);
--
-- Name: purchase_details purchase_details_product_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_details
    ADD CONSTRAINT purchase_details_product_id_foreign FOREIGN KEY (product_id) REFERENCES public.products(id);
--
-- Name: purchase_details purchase_details_purchase_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchase_details
    ADD CONSTRAINT purchase_details_purchase_id_foreign FOREIGN KEY (purchase_id) REFERENCES public.purchases(id);
--
-- Name: purchases purchases_status_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchases
    ADD CONSTRAINT purchases_status_id_foreign FOREIGN KEY (status_id) REFERENCES public.purchase_statuses(id);
--
-- Name: purchases purchases_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.purchases
    ADD CONSTRAINT purchases_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id);
--
-- Name: user_identities user_identities_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id);
--
-- Name: user_roles user_roles_role_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_foreign FOREIGN KEY (role_id) REFERENCES public.roles(id);
--
-- Name: user_roles user_roles_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id);
--
-- Name: users users_prefecture_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_prefecture_id_foreign FOREIGN KEY (prefecture_id) REFERENCES public.prefectures(id);
--
-- PostgreSQL database dump complete
--