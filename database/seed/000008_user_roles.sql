-- 管理者ロール（John Doe のみ。一般ロールと併せ持ち、中間表による複数ロール保持を示す）
INSERT INTO user_roles (user_id, role_id) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'a1b2c3d4-0000-4000-8000-000000000001') ON CONFLICT (user_id, role_id) DO NOTHING;

-- 一般ロール（全サンプルユーザー）
INSERT INTO user_roles (user_id, role_id) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('a95a2dd3-2b37-4def-8041-23d2138faccc', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('0b393ac1-b8a2-4f69-8972-de680aeb0a95', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('090f5b51-37ac-4413-b326-1709ae4661f4', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('d711970c-8e86-4875-8a34-e90bd79096a5', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('211537c3-87ed-4434-af53-676136b35d00', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('e99b0380-522c-4636-a2b6-452acdd7c4ff', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('c688ffbc-731e-4257-82e9-d34b4712afd6', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('c8cc7d69-57aa-44f8-bb07-bf4518cdf98e', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
INSERT INTO user_roles (user_id, role_id) VALUES
('eaabee3e-3b7a-4f61-8fa9-030944625e92', 'a1b2c3d4-0000-4000-8000-000000000002') ON CONFLICT (user_id, role_id) DO NOTHING;
