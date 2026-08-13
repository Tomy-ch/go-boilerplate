-- 追加した一般ユーザー分の mock 認証。JWT 用の subject は mock-auth-server の fixtures に
-- 存在しないため、mock 系だけを登録する。
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000011', 'c23845a3-1bd6-5cc9-9aec-c6e824c65a17', 'mock', 'c23845a3-1bd6-5cc9-9aec-c6e824c65a17') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000012', '65ecbae0-cab1-57c0-9cd0-34699624342e', 'mock', '65ecbae0-cab1-57c0-9cd0-34699624342e') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000013', '3c6b7ebc-983e-518a-bbb9-4e287e18c84e', 'mock', '3c6b7ebc-983e-518a-bbb9-4e287e18c84e') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000014', '1d5b22a9-251d-5317-9bce-2039a364b9a8', 'mock', '1d5b22a9-251d-5317-9bce-2039a364b9a8') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000015', '12821ae8-ec9f-5105-8fee-047008a028bc', 'mock', '12821ae8-ec9f-5105-8fee-047008a028bc') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000016', 'fd8ca491-4ab4-5d97-9dd9-d3ed72b19ed9', 'mock', 'fd8ca491-4ab4-5d97-9dd9-d3ed72b19ed9') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000017', '2825fd84-516e-51c8-b20f-89772f255a1d', 'mock', '2825fd84-516e-51c8-b20f-89772f255a1d') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000018', '239351fa-4bec-556e-9e24-d9337464fe04', 'mock', '239351fa-4bec-556e-9e24-d9337464fe04') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000019', '78444f50-4d29-5fd9-b601-62d1994bb589', 'mock', '78444f50-4d29-5fd9-b601-62d1994bb589') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000020', '4b150976-de4c-59fa-8d79-f43cd2f2cec4', 'mock', '4b150976-de4c-59fa-8d79-f43cd2f2cec4') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000021', 'afce6709-93bd-52ff-af81-0b39c9ea27ed', 'mock', 'afce6709-93bd-52ff-af81-0b39c9ea27ed') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000022', '76f87f1f-657a-5aa6-99e2-16a66edd3725', 'mock', '76f87f1f-657a-5aa6-99e2-16a66edd3725') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000023', '826bbb60-5f49-5560-a6a9-45b61f7351c7', 'mock', '826bbb60-5f49-5560-a6a9-45b61f7351c7') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000024', '18696a36-04a2-5d93-94d0-8e51d3abf45e', 'mock', '18696a36-04a2-5d93-94d0-8e51d3abf45e') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000025', 'e554354a-4f9e-586b-9c2b-3596ca6a7b22', 'mock', 'e554354a-4f9e-586b-9c2b-3596ca6a7b22') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000026', '6ea302d8-254b-56bc-b836-f78cd6059293', 'mock', '6ea302d8-254b-56bc-b836-f78cd6059293') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000027', 'cac76cb4-8939-5f5c-923b-505c2fea1d99', 'mock', 'cac76cb4-8939-5f5c-923b-505c2fea1d99') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000028', 'fdbfb954-15d1-558f-ad5f-5968760d5c86', 'mock', 'fdbfb954-15d1-558f-ad5f-5968760d5c86') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000029', '7aa6051f-cb7c-592d-baf6-1fd93ca20488', 'mock', '7aa6051f-cb7c-592d-baf6-1fd93ca20488') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000030', '19411347-79f7-5477-bc01-165f8ce1fdbb', 'mock', '19411347-79f7-5477-bc01-165f8ce1fdbb') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000031', '099bfa4f-4e6f-586d-af12-9bfce82cc552', 'mock', '099bfa4f-4e6f-586d-af12-9bfce82cc552') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000032', '315f985b-cccc-50a3-b084-6be7f9a49a03', 'mock', '315f985b-cccc-50a3-b084-6be7f9a49a03') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000033', '8fda6537-8ae0-584c-aefa-211124150bf5', 'mock', '8fda6537-8ae0-584c-aefa-211124150bf5') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000034', '442f24ed-a54e-5ec8-882c-ebee3fbdb071', 'mock', '442f24ed-a54e-5ec8-882c-ebee3fbdb071') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000035', '589dd2fc-09e0-55d2-83d9-93bc93719f49', 'mock', '589dd2fc-09e0-55d2-83d9-93bc93719f49') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000036', 'bb4a53ae-7064-515f-b795-16c9544c8bae', 'mock', 'bb4a53ae-7064-515f-b795-16c9544c8bae') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000037', '6d24fed9-a0f6-57b2-9213-d18e25697f11', 'mock', '6d24fed9-a0f6-57b2-9213-d18e25697f11') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000038', '94e379c1-2899-5023-9edd-7acbce6cacb1', 'mock', '94e379c1-2899-5023-9edd-7acbce6cacb1') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000039', '8502e70f-40a5-5868-ab51-c9d97bcb95c4', 'mock', '8502e70f-40a5-5868-ab51-c9d97bcb95c4') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000040', 'caa57b5c-91bb-52e9-972c-911dcdacc295', 'mock', 'caa57b5c-91bb-52e9-972c-911dcdacc295') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000041', '71f89245-d449-5ddb-a98f-3a806bce4a9a', 'mock', '71f89245-d449-5ddb-a98f-3a806bce4a9a') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000042', '5c9b4068-c5e9-557d-bd64-7b8db01082af', 'mock', '5c9b4068-c5e9-557d-bd64-7b8db01082af') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000043', '7995788f-c550-5493-b580-90005e775170', 'mock', '7995788f-c550-5493-b580-90005e775170') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000044', '3bff4778-3d35-5d80-bb55-9f18235eb966', 'mock', '3bff4778-3d35-5d80-bb55-9f18235eb966') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000045', '9fa5e3e3-2c60-5402-b938-2eb1f731227a', 'mock', '9fa5e3e3-2c60-5402-b938-2eb1f731227a') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000046', 'ed9927ae-262e-5371-acad-dd513208afe2', 'mock', 'ed9927ae-262e-5371-acad-dd513208afe2') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000047', 'b974d326-705d-5662-9feb-f32d83f7dc4c', 'mock', 'b974d326-705d-5662-9feb-f32d83f7dc4c') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000048', '857ef3b9-8241-57b7-ad25-7fb015dc41a8', 'mock', '857ef3b9-8241-57b7-ad25-7fb015dc41a8') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000049', '8bb1ee74-4950-565c-892f-20f5b7f5c2c1', 'mock', '8bb1ee74-4950-565c-892f-20f5b7f5c2c1') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_identities (id, user_id, issuer, subject) VALUES
('d1b2c3d4-0000-4000-8000-000000000050', '958fe254-92fc-5177-aed0-b584e79d1c8c', 'mock', '958fe254-92fc-5177-aed0-b584e79d1c8c') ON CONFLICT (id) DO NOTHING;
