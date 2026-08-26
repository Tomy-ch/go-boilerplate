CREATE TABLE IF NOT EXISTS product_statuses (
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    code SMALLINT NOT NULL,
    sort_key SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_statuses_id_primary PRIMARY KEY (id),
    CONSTRAINT product_statuses_name_unique UNIQUE (name),
    CONSTRAINT product_statuses_code_unique UNIQUE (code),
    CONSTRAINT product_statuses_sort_key_unique UNIQUE (sort_key)
);

COMMENT ON TABLE product_statuses IS '商品ステータス';
COMMENT ON COLUMN product_statuses.id IS 'ID';
COMMENT ON COLUMN product_statuses.name IS '名称';
COMMENT ON COLUMN product_statuses.code IS 'コード';
COMMENT ON COLUMN product_statuses.sort_key IS '順序';
COMMENT ON COLUMN product_statuses.created_at IS '作成日時';
COMMENT ON COLUMN product_statuses.updated_at IS '更新日時';

INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('093170fb-83a2-4864-a2b3-53236eaf3597', '在庫あり', 1, 5) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('f33654fe-1041-498d-be18-3a1384c10df4', '在庫切れ', 2, 6) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('cf02b28b-b300-446a-a7df-d85c26fa991e', '予約受付中', 3, 4) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('4a101028-0abb-464f-a12d-fd8a1fada472', '販売終了', 4, 7) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('d8002e6c-21b5-41e6-94a6-7703f013a0f2', '取り寄せ中', 5, 3) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('300ed420-eb71-4029-bf29-a7182421a0c2', '入荷待ち', 6, 2) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('14b21a78-e6f5-48d9-8222-c4db690a6e52', '廃盤', 7, 9) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('bdf44f06-227c-4549-b2c8-e57b32f06321', '検討中', 8, 1) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('96b9a6f1-d2b3-477a-90f4-cc8d00871429', '再入荷予定', 9, 8) ON CONFLICT (id) DO NOTHING;
INSERT INTO product_statuses (id, name, code, sort_key) VALUES
('008bdc95-311a-4b52-b3f2-5956fc5995ee', '限定販売', 10, 10) ON CONFLICT (id) DO NOTHING;
