-- 外部ID連携（issuer + subject → 内部ユーザー）。IdentityResolver が (issuer, subject) で解決する。
-- mock 系: ローカル開発用（issuer=mock）。subject は内部ユーザーの UUID と一致させる。
-- jwt 系: JWT 認証向け。`${AUTH_ISSUER}` は投入時に seed ランナーがその環境の issuer へ展開する（worktree のスロットでポートがずれるため直書きしない）。
-- subject は mock-auth-server の fixtures（mock-auth-server/fixtures/users.json）と一致させる。
-- jwt 系だけ再投入で issuer を更新する。DO NOTHING だと別環境で投入済みの DB に古い issuer が残り、認証だけが通らない状態になる。
-- 削除済みユーザー（Charlie / Frank）も登録し、状態検証で認証失敗することを確認できるようにする。unknown-user は行を作らず未登録失敗を表す。

-- mock 認証用（issuer=mock, subject=内部UUID）
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000001', '550e8400-e29b-41d4-a716-446655440000', 'mock', '550e8400-e29b-41d4-a716-446655440000') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000002', 'a95a2dd3-2b37-4def-8041-23d2138faccc', 'mock', 'a95a2dd3-2b37-4def-8041-23d2138faccc') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000003', '0b393ac1-b8a2-4f69-8972-de680aeb0a95', 'mock', '0b393ac1-b8a2-4f69-8972-de680aeb0a95') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000004', '090f5b51-37ac-4413-b326-1709ae4661f4', 'mock', '090f5b51-37ac-4413-b326-1709ae4661f4') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000005', 'd711970c-8e86-4875-8a34-e90bd79096a5', 'mock', 'd711970c-8e86-4875-8a34-e90bd79096a5') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000006', '211537c3-87ed-4434-af53-676136b35d00', 'mock', '211537c3-87ed-4434-af53-676136b35d00') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000007', 'e99b0380-522c-4636-a2b6-452acdd7c4ff', 'mock', 'e99b0380-522c-4636-a2b6-452acdd7c4ff') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000008', 'c688ffbc-731e-4257-82e9-d34b4712afd6', 'mock', 'c688ffbc-731e-4257-82e9-d34b4712afd6') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000009', 'c8cc7d69-57aa-44f8-bb07-bf4518cdf98e', 'mock', 'c8cc7d69-57aa-44f8-bb07-bf4518cdf98e') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000010', 'eaabee3e-3b7a-4f61-8fa9-030944625e92', 'mock', 'eaabee3e-3b7a-4f61-8fa9-030944625e92') ON CONFLICT (id) DO NOTHING;

-- JWT 認証用（issuer=${AUTH_ISSUER}, subject=fixtures）
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000001', '550e8400-e29b-41d4-a716-446655440000', '${AUTH_ISSUER}', 'user-john-doe') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000002', 'a95a2dd3-2b37-4def-8041-23d2138faccc', '${AUTH_ISSUER}', 'user-jane-smith') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000003', '0b393ac1-b8a2-4f69-8972-de680aeb0a95', '${AUTH_ISSUER}', 'user-alice-johnson') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000004', '090f5b51-37ac-4413-b326-1709ae4661f4', '${AUTH_ISSUER}', 'user-bob-brown') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000005', 'd711970c-8e86-4875-8a34-e90bd79096a5', '${AUTH_ISSUER}', 'user-charlie-davis') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000006', '211537c3-87ed-4434-af53-676136b35d00', '${AUTH_ISSUER}', 'user-eve-white') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000007', 'e99b0380-522c-4636-a2b6-452acdd7c4ff', '${AUTH_ISSUER}', 'user-frank-green') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000008', 'c688ffbc-731e-4257-82e9-d34b4712afd6', '${AUTH_ISSUER}', 'user-grace-lee') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000009', 'c8cc7d69-57aa-44f8-bb07-bf4518cdf98e', '${AUTH_ISSUER}', 'user-hank-wilson') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('e1b2c3d4-0000-4000-8000-000000000010', 'eaabee3e-3b7a-4f61-8fa9-030944625e92', '${AUTH_ISSUER}', 'user-ivy-clark') ON CONFLICT (id) DO UPDATE SET issuer = excluded.issuer, updated_at = NOW();
