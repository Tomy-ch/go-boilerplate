-- 状態機械 PATCH（pay / ship / deliver）の遷移先ステータスを追加する。
-- seed の 6 種（未処理〜キャンセル）には「支払い済み」「発送済み」「配達済み」が無く、
-- pay の遷移先 status_id が確定しないため FK 違反になる。pay / ship / deliver で個別採番せず
-- 本 migration にまとめて追加する（ADR-0024 append-only / ADR-0026 master-via-migration）。
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
('4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c', '支払い済み', 7, 7) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
('5c9a1f3b-2d4e-4b6f-9c8a-3e0d1f2b4c5d', '発送済み', 8, 8) ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES
('6d0b2a4c-3e5f-4c7a-ad9b-4f1e2a3c5d6e', '配達済み', 9, 9) ON CONFLICT (id) DO NOTHING;
