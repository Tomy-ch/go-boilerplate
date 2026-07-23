INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'John', 'Doe', 'john.doe@example.com', '123-456-7890', 'faba7bb2-f5a0-4a51-adae-1564929077b2', '札幌', '1-1', 'Building A', '060-0001', '2022-01-01T00:00:00', '2023-01-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('a95a2dd3-2b37-4def-8041-23d2138faccc', 'Jane', 'Smith', 'jane.smith@example.com', '098-765-4321', '705349ae-d6d8-48ad-8263-6ab48cc9201b', '青森', '2-2', 'Building B', '030-0002', '2023-07-01T00:00:00', '2024-01-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('0b393ac1-b8a2-4f69-8972-de680aeb0a95', 'Alice', 'Johnson', 'alice.johnson@example.com', '555-555-5555', '212f525e-ffab-4523-9ec0-8af76e006fe3', '盛岡', '3-3', 'Building C', '020-0003', '2023-01-01T00:00:00', '2023-07-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) VALUES
('090f5b51-37ac-4413-b326-1709ae4661f4', 'Bob', 'Brown', 'bob.brown@example.com', '444-444-4444', '101caa1e-84e7-4ceb-9108-50d40b6be1a3', '千代田区', '1-1-1', '001-0101', '2024-01-01T00:00:00', '2024-07-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, deleted_at, created_at, updated_at) VALUES
('d711970c-8e86-4875-8a34-e90bd79096a5', 'Charlie', 'Davis', 'charlie.davis@example.com', '333-333-3333', '101caa1e-84e7-4ceb-9108-50d40b6be1a3', '中央区', '2-2-2', 'Building D', '104-0061', '2025-01-01T00:00:00', '2024-07-01T00:00:00', '2024-12-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('211537c3-87ed-4434-af53-676136b35d00', 'Eve', 'White', 'eve.white@example.com', '222-222-2222', '101caa1e-84e7-4ceb-9108-50d40b6be1a3', '港区', '3-3-3', 'Building E', '105-0003', '2022-07-01T00:00:00', '2024-09-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, deleted_at, created_at, updated_at) VALUES
('e99b0380-522c-4636-a2b6-452acdd7c4ff', 'Frank', 'Green', 'frank.green@example.com', '111-111-1111', '101caa1e-84e7-4ceb-9108-50d40b6be1a3', '新宿区', '4-4-4', '160-0004', '2025-03-01T00:00:00', '2023-11-01T00:00:00', '2024-07-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('c688ffbc-731e-4257-82e9-d34b4712afd6', 'Grace', 'Lee', 'grace.lee@example.com', '000-000-0000', 'd647fc85-ff46-4530-88cb-198f4a68a9d7', '大阪市', '5-5-5', 'Building F', '530-0001', '2024-11-01T00:00:00', '2024-12-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) VALUES
('c8cc7d69-57aa-44f8-bb07-bf4518cdf98e', 'Hank', 'Wilson', 'hank.wilson@example.com', '999-999-9999', '71e69706-5c1a-4cef-a7cb-c44b8b79c89b', '京都市', '6-6-6', '600-0001', '2024-12-01T00:00:00', '2025-07-01T00:00:00') ON CONFLICT (id) DO NOTHING;
INSERT INTO users (id, first_name, last_name, email, phone, prefecture_id, city, street, building, postal_code, created_at, updated_at) VALUES
('eaabee3e-3b7a-4f61-8fa9-030944625e92', 'Ivy', 'Clark', 'ivy.clark@example.com', '888-888-8888', 'a03aaec4-3bd6-4bfb-8e47-2fbfa026d344', '鹿児島市', '7-7-7', 'Building G', '890-0001', '2025-04-01T00:00:00', '2025-06-01T00:00:00') ON CONFLICT (id) DO NOTHING;
