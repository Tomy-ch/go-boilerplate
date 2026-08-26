INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3a8ac9ac-36cb-55a6-a9d5-78171bfddcba', '55c5b5eb-e973-58a2-9fa9-f86b909560a7', '2122ae16-432c-59c4-ab5b-e9d5eddcf3fe', 1, '19.07', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0e4b7234-d41a-5240-8d27-80ff4f80e327', '55c5b5eb-e973-58a2-9fa9-f86b909560a7', 'cf50b9ea-37ae-5a16-b610-22ae0d214e1f', 2, '8.00', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('893f79bf-7ac4-5533-bf15-059c646c6a8d', '55c5b5eb-e973-58a2-9fa9-f86b909560a7', '9ba40a67-5194-4777-a16e-fb5ffc706a26', 3, '23.33', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('efe29e87-c317-5ce3-80b2-a0f45b91afdc', '47e0fdbc-a529-5059-a10a-2777415fabfa', 'e78002a8-dd64-5c20-bfd7-25c521720539', 1, '23.47', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('80b400b6-8bf0-5cc4-8fdf-f35ea547c476', 'f9a12bb4-056f-5220-8beb-6caed0031727', 'db5a1adf-4fcb-53cd-9047-3ee188ceb856', 2, '20.53', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('b65278b2-7bc7-5d64-b208-3472f5db0d86', 'f9a12bb4-056f-5220-8beb-6caed0031727', '54cae87a-128d-5255-a620-2d84ec385040', 3, '13.93', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('d306e426-72e7-5dce-b4a3-90965012efd0', '0bd42560-a297-5400-8126-5b1f3eed48c4', 'a6f9b205-a205-5def-86ae-0b72ab61bb2b', 3, '17.31', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('d4e669fd-d9ec-5036-8483-318d6ebc6fdf', '0bd42560-a297-5400-8126-5b1f3eed48c4', '445a017a-4273-5030-be79-e7cf173d8628', 1, '13.20', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('1e5e1bc8-8e58-5904-9265-82e8c08bad94', '0bd42560-a297-5400-8126-5b1f3eed48c4', 'd034c308-48da-463c-977d-1a613c954df2', 2, '10.00', NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e26f53f3-5418-546c-97bf-5bccaf8ae9e2', '4c33f129-e66f-5fc5-b07a-592b4df81b04', 'f3a4e426-6f0c-4a4e-a452-6c71dd7b1373', 2, '4.67', NOW() - INTERVAL '426 days', NOW() - INTERVAL '426 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3caa697a-ece7-50ad-b421-5b6e3f1957c5', '4c33f129-e66f-5fc5-b07a-592b4df81b04', '8f32f29c-154e-5e41-b5d8-5718f9a89c92', 3, '8.53', NOW() - INTERVAL '426 days', NOW() - INTERVAL '426 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('72c3db09-87c7-5565-948a-d073f97120c3', '4c33f129-e66f-5fc5-b07a-592b4df81b04', '90bff965-7e2c-5a93-8fab-a8a98897b423', 1, '8.67', NOW() - INTERVAL '426 days', NOW() - INTERVAL '426 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('48349207-b5e3-5a0b-ad35-a7f37b298079', '83f1f2e9-5608-5153-80bf-bad56af33a7d', '6de17933-7149-43b1-9aac-83e1b68fb85d', 3, '3.33', NOW() - INTERVAL '427 days', NOW() - INTERVAL '427 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('f38733bf-1d42-5350-a010-1a9da3d3932b', '83f1f2e9-5608-5153-80bf-bad56af33a7d', 'f5bdf33d-ca32-5e98-8602-cd1f2ae7b01f', 1, '15.20', NOW() - INTERVAL '427 days', NOW() - INTERVAL '427 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('c6cf1d47-f6a7-5918-aa71-197df5a25271', '83f1f2e9-5608-5153-80bf-bad56af33a7d', 'c53f7ffe-0126-535d-824e-5dde09f953f6', 2, '3.32', NOW() - INTERVAL '427 days', NOW() - INTERVAL '427 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('9e73e1dd-6d6b-5b64-b79e-c7615300474b', '83f1f2e9-5608-5153-80bf-bad56af33a7d', '9e1f070e-3eab-5d6d-af35-b364b1fc8947', 3, '5.20', NOW() - INTERVAL '427 days', NOW() - INTERVAL '427 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e58ff898-4679-549e-8c96-a0cc1e3938b6', 'abc65282-e469-58df-b265-1a5a83db32b3', '5555fc9d-dd5c-4f71-a611-8f89c4de02cc', 1, '2.67', NOW() - INTERVAL '428 days', NOW() - INTERVAL '428 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('429eaabc-51bd-5b0d-bc02-ebc3f3f381ad', '4d813ff9-49b5-5471-a5c1-0fb7347e6024', '16469cbb-e84d-456f-85b9-0ba1c41a8349', 2, '4.00', NOW() - INTERVAL '429 days', NOW() - INTERVAL '429 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e59f8436-f3d4-52f6-ba2c-e48e5bebffe7', '4d813ff9-49b5-5471-a5c1-0fb7347e6024', 'e2d88606-cb28-5869-8788-70192050b6c0', 3, '10.53', NOW() - INTERVAL '429 days', NOW() - INTERVAL '429 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0d24f799-99e7-57ab-8918-a833452af57b', 'ee8bed4d-2c34-5896-bb9c-498f211844d0', 'a5a34193-d84e-46c9-9302-b6056c5b0c5f', 3, '4.67', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('b35f9404-8be4-53e7-b914-19aef4509cf2', 'ee8bed4d-2c34-5896-bb9c-498f211844d0', '265d45d2-d674-5226-8b92-79953fa756de', 1, '14.40', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a50498bc-8b94-5745-a76b-8f739dac2502', 'ee8bed4d-2c34-5896-bb9c-498f211844d0', '575fdd08-84ca-51ab-821d-2da59344ef3d', 2, '11.00', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('2992b401-9654-5312-a7f3-f09c516b0f84', 'ef78edac-7e9e-57f2-b6a6-72dbf8af8775', '1297ad91-c870-561f-982b-332d1bad1f37', 1, '15.20', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('c31c39d7-707c-5b23-b6d5-c64336525c9c', 'ef78edac-7e9e-57f2-b6a6-72dbf8af8775', '408a545b-ed47-5f7d-89f4-74c9fa6c67a4', 2, '15.87', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('da3b50c1-3e36-5c2d-bfad-b9c554db290d', 'ef78edac-7e9e-57f2-b6a6-72dbf8af8775', '5576c3c3-f474-567b-9c74-3f8f16639283', 3, '8.53', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('004747dd-40dd-5029-9b8d-19996eacdfa5', 'ef78edac-7e9e-57f2-b6a6-72dbf8af8775', 'a76f37aa-b93b-5d36-a38a-85063a47b770', 1, '6.53', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0b70b012-d23f-58c5-8753-040bc13025b4', '89dc01f8-1bac-5542-a3f2-d0e4e8f33096', '9cba6803-642c-5a5b-a186-d20b34d26b02', 3, '1.99', NOW() - INTERVAL '374 days', NOW() - INTERVAL '374 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('927315ca-ec89-5109-821e-12b46551eea0', '89dc01f8-1bac-5542-a3f2-d0e4e8f33096', 'e0a87f27-1eac-549a-9ed1-1a519c616d91', 1, '2.19', NOW() - INTERVAL '374 days', NOW() - INTERVAL '374 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('ad3cb7c9-64b7-5fb5-bf3c-a4ae90bc8e19', '89dc01f8-1bac-5542-a3f2-d0e4e8f33096', '6d60a1ca-4b0f-56e5-9a88-25f8ecc52d64', 2, '2.19', NOW() - INTERVAL '374 days', NOW() - INTERVAL '374 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('becf68f7-415f-5522-aa31-2ea7a9dea955', 'c1a6c8bf-76c6-572b-8f91-c5726eb89bce', 'f59a52e0-4bae-5400-aa0d-23dac6f8a569', 1, '2.19', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('95ccac00-c472-5883-8283-7471761f06f7', 'c1a6c8bf-76c6-572b-8f91-c5726eb89bce', '6495a0dc-28c1-5227-83ba-3bd0288a168b', 2, '6.65', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3d3274d0-082f-5413-9bdb-6f10cbadf7fc', 'c1a6c8bf-76c6-572b-8f91-c5726eb89bce', '5675b3d9-aebe-5025-a726-9f1a3f5b4f55', 3, '5.99', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e2f17c32-d8aa-5499-b57d-6f9c19acd470', 'c1a6c8bf-76c6-572b-8f91-c5726eb89bce', '4f29a84e-7197-51df-bcb8-cf4e63c42dd6', 1, '4.65', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('23767971-442d-5148-abf5-99068cf95f8f', 'b0ae3205-f85d-5ce3-a9ff-b8d6a34a763c', 'd1f53065-325a-5b9b-b6c5-75fc5e548398', 2, '2.85', NOW() - INTERVAL '376 days', NOW() - INTERVAL '376 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3419dc8b-6625-56f9-bb2a-2925ed84723b', '8c656d3b-6136-59e0-8968-ac0dfab5fdde', '200c10c4-3fce-5e62-aea9-5d1c66faa927', 3, '158.67', NOW() - INTERVAL '292 days', NOW() - INTERVAL '292 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('83119d4a-c4ab-5f4f-a760-34e7d3a683b5', 'e851873b-287e-5bdf-b1b4-4e062e7ed8f1', '00a4c85e-6d90-5567-9f4d-e7ab46d51a74', 1, '332.00', NOW() - INTERVAL '293 days', NOW() - INTERVAL '293 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0c0d1a3d-3f03-5536-ac8e-3720bbbf88ba', 'e851873b-287e-5bdf-b1b4-4e062e7ed8f1', '21e3cb7c-95f7-4c1b-9127-b9852bd37fc3', 2, '10.00', NOW() - INTERVAL '293 days', NOW() - INTERVAL '293 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a1396fc9-f295-5494-92fc-172be6e77b7d', '22805716-355b-50ce-bae2-716f46d5fbff', 'cf10e51e-bfea-591e-a2b0-96a0107e4859', 2, '132.00', NOW() - INTERVAL '294 days', NOW() - INTERVAL '294 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('fb9da39e-c82d-534b-933b-476743777bfe', '22805716-355b-50ce-bae2-716f46d5fbff', 'f9582be9-639f-443e-bcc8-c722ec68f3be', 3, '16.67', NOW() - INTERVAL '294 days', NOW() - INTERVAL '294 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0456441b-0f5f-5c96-a2a2-64da91a98543', '22805716-355b-50ce-bae2-716f46d5fbff', 'db5a1adf-4fcb-53cd-9047-3ee188ceb856', 1, '20.53', NOW() - INTERVAL '294 days', NOW() - INTERVAL '294 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('4c8ea18f-17a6-59e0-8ce5-1fc085cef1fe', '4b95200e-4181-5a3b-bdc7-63339c0fbc8f', '2321ef33-ce9c-595b-aeef-4793439df397', 3, '105.33', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('624270a7-67c5-5e40-ad32-d5891f5b5f14', '4b95200e-4181-5a3b-bdc7-63339c0fbc8f', '208f335c-9932-52e8-b67f-1d24ffadebd1', 1, '35.20', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('96de9407-0bf4-5552-ad01-3f15dcd1c000', '4b95200e-4181-5a3b-bdc7-63339c0fbc8f', 'ead60865-1de1-5466-8d2b-66c51da66f88', 2, '27.87', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('819a529d-903b-5cd5-8d60-979e2bd10e9a', '4b95200e-4181-5a3b-bdc7-63339c0fbc8f', '9c605fee-d1c2-54ee-836e-26ba32394950', 3, '19.65', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('348e6ed7-4a02-5712-8342-5a777e1e7d99', 'f4a35eb6-2473-53c1-ae6f-d117c8577fd0', '760c1825-6aac-4d33-8887-861985d8204c', 2, '6.00', NOW() - INTERVAL '163 days', NOW() - INTERVAL '163 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('c89a4950-048b-5655-a2b1-c7f5b4f808f5', 'f4a35eb6-2473-53c1-ae6f-d117c8577fd0', '32587b37-56f0-475f-bc21-9c9265efc7d6', 3, '6.67', NOW() - INTERVAL '163 days', NOW() - INTERVAL '163 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('597fa1ac-0a4d-5f38-b769-f1d8996d80ed', 'f4a35eb6-2473-53c1-ae6f-d117c8577fd0', '6e5d8b87-7087-4cb1-9a28-fe6481f356e1', 1, '5.33', NOW() - INTERVAL '163 days', NOW() - INTERVAL '163 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('b34779d4-57c7-52bd-954f-caa2869dc301', 'f4a35eb6-2473-53c1-ae6f-d117c8577fd0', '7681e3c6-52bc-4c5f-b03f-b054c2611b31', 2, '4.00', NOW() - INTERVAL '163 days', NOW() - INTERVAL '163 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a8a65c9b-0716-5247-86b5-10763d81c94b', '5d88cc3d-41d0-522c-98c1-c74192838053', 'a586d01b-e4f3-4962-af17-789b7e383cf8', 3, '3.33', NOW() - INTERVAL '164 days', NOW() - INTERVAL '164 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3edffd4f-f671-561a-a749-4d2873111e5b', '1c8841aa-daf7-51ed-8f9a-2f7d449a215c', 'a1854d1e-5abd-42e3-847a-5e911f022010', 1, '2.00', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('8aba3df2-a860-58a8-917c-c7bf0d04da2b', '1c8841aa-daf7-51ed-8f9a-2f7d449a215c', '807cbba9-63fe-489b-a451-fbde8d39f112', 2, '4.00', NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('5faf646d-5ca0-58cd-a68b-886b4336ab3a', '09dc182e-5361-5f88-bd71-e982e5694bc1', '506dd665-9375-42b9-8b24-03ce002ab7de', 2, '2.67', NOW() - INTERVAL '166 days', NOW() - INTERVAL '166 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('516b92ed-ab6c-5a28-ad34-b9b7a2d5e482', '09dc182e-5361-5f88-bd71-e982e5694bc1', '41bcb0f4-6bca-45ee-8c50-aefff0fce62e', 3, '2.67', NOW() - INTERVAL '166 days', NOW() - INTERVAL '166 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('1a8446e2-a0a0-50b2-98cc-198512be7c8d', '09dc182e-5361-5f88-bd71-e982e5694bc1', 'a8e61137-c458-4715-8a04-95148f2e814b', 1, '3.33', NOW() - INTERVAL '166 days', NOW() - INTERVAL '166 days') ON CONFLICT (id) DO NOTHING;
