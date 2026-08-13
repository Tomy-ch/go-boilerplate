CREATE TABLE IF NOT EXISTS prefectures (
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    code SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT prefectures_id_primary PRIMARY KEY (id),
    CONSTRAINT prefectures_name_unique UNIQUE (name),
    CONSTRAINT prefectures_code_unique UNIQUE (code)
);

COMMENT ON TABLE prefectures IS '都道府県';
COMMENT ON COLUMN prefectures.id IS 'ID';
COMMENT ON COLUMN prefectures.name IS '都道府県名';
COMMENT ON COLUMN prefectures.code IS '都道府県コード';
COMMENT ON COLUMN prefectures.created_at IS '作成日時';
COMMENT ON COLUMN prefectures.updated_at IS '更新日時';

INSERT INTO prefectures (id, name, code) VALUES (
    'faba7bb2-f5a0-4a51-adae-1564929077b2', '北海道', 1
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '705349ae-d6d8-48ad-8263-6ab48cc9201b', '青森県', 2
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '212f525e-ffab-4523-9ec0-8af76e006fe3', '岩手県', 3
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'd731fb02-faaa-4cb0-a926-d62ed57e3e80', '宮城県', 4
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '6a1bade8-f045-40d0-8add-79c423bd4b6d', '秋田県', 5
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '789acf37-4099-48d9-9131-da685d5aa65c', '山形県', 6
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '325640f7-3297-459a-9233-d3661a259cd6', '福島県', 7
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '101caa1e-84e7-4ceb-9108-50d40b6be1a3', '東京都', 8
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '08787d6f-9a19-46ad-aaa1-da3e369c343b', '神奈川県', 9
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '81fd5704-50a9-4c1b-8f92-799ff33c2b70', '千葉県', 10
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '30810788-bc8b-4828-a8b8-0a249abd3a41', '埼玉県', 11
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'e1b5b55f-181b-47d0-afd0-92f2bb6e437b', '茨城県', 12
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'b91dad53-547a-444f-a02c-8919ef91f759', '栃木県', 13
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'e38b17a2-76b9-4f13-987b-b89376adcf0b', '群馬県', 14
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '0e182d9a-1626-488f-9706-a2b3a0da6433', '新潟県', 15
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'b059e017-031d-4a3b-a316-3aba80ce4954', '富山県', 16
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '5dace767-df5f-4751-82fd-0324862a0900', '石川県', 17
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'ac7ecc45-1af8-4f9d-b36f-aef4cb199a4c', '福井県', 18
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'ae92457f-5dd7-4e0a-b3c8-220ed572014a', '山梨県', 19
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '2bb2ec61-2276-4ffd-b231-1fea95041d3b', '長野県', 20
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '0775fe11-df27-4488-92de-018b4fae66b1', '岐阜県', 21
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '3804d878-7624-45be-b3e9-b75a17f8ba78', '静岡県', 22
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '5418e9ff-1e62-44e0-a406-0a3bea37ce1d', '愛知県', 23
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '24e59f84-7dab-4581-9a6e-53638dd9a1b3', '三重県', 24
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '7d22118d-39df-41d5-beee-53af6b4feacf', '滋賀県', 25
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '71e69706-5c1a-4cef-a7cb-c44b8b79c89b', '京都府', 26
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'd647fc85-ff46-4530-88cb-198f4a68a9d7', '大阪府', 27
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '0f8536a1-c1db-4d14-8d40-2e3f18b55b0d', '兵庫県', 28
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'e60e8b94-8505-4fef-ab41-8712860a08f7', '奈良県', 29
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '52a4b2ce-7045-4ff3-a223-65fe9123fd3a', '和歌山県', 30
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '001a0638-2590-4c9a-be4e-c60f9f65e8b9', '鳥取県', 31
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'd80dc53d-5df0-4f7e-8dec-67deb62ff65e', '島根県', 32
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '2bae289c-3083-429d-8bb3-d5d5b2af5510', '岡山県', 33
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'e3a32f7e-9625-4f8a-b5cd-00683c03b4e0', '広島県', 34
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '6e0e2e41-691a-4188-8bf6-5ee9f5e9118b', '山口県', 35
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '8acae20f-7259-4657-8b24-f889e18a8ed6', '徳島県', 36
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'a2027e6f-ae69-4438-b4bd-ee76b1d68073', '香川県', 37
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '7c33e32c-83cc-4719-a22c-5a867ade29e1', '愛媛県', 38
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '95511bd2-50d7-406b-a2f3-ff2775791e40', '高知県', 39
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '058026a6-82d9-4538-9f45-e18a3cd8c99a', '福岡県', 40
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '8b304c60-aed8-4936-8f51-1aa4fc4a3316', '佐賀県', 41
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '73ec142b-8b46-4d8c-880f-04c1df1ae009', '長崎県', 42
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '70930f26-e400-4a64-a328-963f2f16b062', '熊本県', 43
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '89ce0b0b-4cba-4681-816f-d2320ca79879', '大分県', 44
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '43e66f28-104f-4106-8cbe-be7dc100d43d', '宮崎県', 45
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    'a03aaec4-3bd6-4bfb-8e47-2fbfa026d344', '鹿児島県', 46
) ON CONFLICT (id) DO NOTHING;
INSERT INTO prefectures (id, name, code) VALUES (
    '55325af0-f741-44dd-b918-a51452a202fe', '沖縄県', 47
) ON CONFLICT (id) DO NOTHING;
