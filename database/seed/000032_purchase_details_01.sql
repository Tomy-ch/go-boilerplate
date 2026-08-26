-- 購入明細。単価は購入時点の商品価格（USD ドル）を写した値。
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('80966c99-7d54-5194-99b8-782b212a05a5', 'b6636865-c8e4-5c65-ba03-f372e0f020c9', '12d69405-69ae-53d4-9227-047ca671f28b', 2, '55.00', NOW() - INTERVAL '428 days', NOW() - INTERVAL '428 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('2e271362-365d-5972-a9ef-dc43c8c5f4cc', 'c11c0429-c505-56c8-aac3-ef1b04ea8ae8', '0fa3a079-7669-575c-baaf-aa7aba4e945e', 3, '80.67', NOW() - INTERVAL '429 days', NOW() - INTERVAL '429 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('d80da1bc-8807-58e7-b141-71de1c935c25', 'c11c0429-c505-56c8-aac3-ef1b04ea8ae8', '6be9ec15-dc7d-5eaa-9ed6-9f26d25812a3', 1, '66.00', NOW() - INTERVAL '429 days', NOW() - INTERVAL '429 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('655e065b-4672-5c15-ad17-3d17c8ad5505', '497317b8-973b-57bb-80ec-da87d153b771', '6143dc48-9ea3-52da-a507-e2d5172c525b', 1, '205.33', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('9a124fe4-ede7-52df-ac9d-b72d2702ecb5', '497317b8-973b-57bb-80ec-da87d153b771', '39ce04df-1b91-55dc-b232-d4fafe0d3855', 2, '8.60', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a395c095-b96d-5bc6-88f1-e89ee1c68e0d', '497317b8-973b-57bb-80ec-da87d153b771', '07aca1f7-3d1b-4ead-8cf1-80ca0fe4208f', 3, '2.00', NOW() - INTERVAL '430 days', NOW() - INTERVAL '430 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0331e63b-2b7f-531e-a6f6-0689d2499636', '889b70d1-8548-5c18-b17f-02f6b1e80b88', '6f684364-f423-5afc-835b-755b65b49d8e', 2, '124.67', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('530ce3c8-f393-5ae8-85dc-24817cf61fe5', '889b70d1-8548-5c18-b17f-02f6b1e80b88', '1ca8662f-1cc7-5de4-baad-82955c8cc4c0', 3, '44.00', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('90fb60c8-dd7d-512f-a862-e597fca429ca', '889b70d1-8548-5c18-b17f-02f6b1e80b88', '6f80f9f7-0963-544d-8745-d1e9824654c4', 1, '146.67', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('80860ff3-7c5e-5585-852f-e714f47899cf', '889b70d1-8548-5c18-b17f-02f6b1e80b88', '588aee99-26c4-5f10-93c8-cb2698d8a877', 2, '396.00', NOW() - INTERVAL '431 days', NOW() - INTERVAL '431 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('70e0f8fe-3254-56d3-b9d0-15c02a234d80', 'e00e052d-28e8-5f03-ba89-1c9653972849', 'a421ef50-d358-5a50-9387-7822ff2b5c7d', 1, '21.20', NOW() - INTERVAL '512 days', NOW() - INTERVAL '512 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('98735169-9362-5322-9d2c-c25b1878ccbb', '211f6253-673b-52f9-a803-9bdf971299a8', '9e1f070e-3eab-5d6d-af35-b364b1fc8947', 2, '5.20', NOW() - INTERVAL '513 days', NOW() - INTERVAL '513 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e2bf84af-6627-5e14-96c4-c134410750de', '211f6253-673b-52f9-a803-9bdf971299a8', '1cb0d25f-37f8-59e0-b7c6-d3f2abdacf11', 3, '2.85', NOW() - INTERVAL '513 days', NOW() - INTERVAL '513 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e05b5ca8-defe-5260-80ba-2cc9c99d5cd8', '99c5a687-9c6b-524c-bad3-644f06aa546d', '3e5e31f4-6476-54e5-bdd0-80aad6be3791', 3, '19.87', NOW() - INTERVAL '514 days', NOW() - INTERVAL '514 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('8f09bef6-a8fa-557c-bd1f-9cd8bb5dd2ae', '99c5a687-9c6b-524c-bad3-644f06aa546d', 'fa0a3868-027c-588c-90f5-ed974f64e804', 1, '2.85', NOW() - INTERVAL '514 days', NOW() - INTERVAL '514 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('15b4821a-733a-528b-bbde-5b8c4f648fcf', '99c5a687-9c6b-524c-bad3-644f06aa546d', 'de86f418-0345-5d5e-80d4-5bfeb19c5f13', 2, '2.65', NOW() - INTERVAL '514 days', NOW() - INTERVAL '514 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('ed0b9c10-8d57-552b-a177-a553a71c6f3f', 'ed31d490-cc0b-59b1-a18f-107e60a46bbf', '2e944a4c-ad40-55e7-b513-4b191ab3d1e3', 1, '3.32', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('9d435eb0-3cc1-5a43-bf5f-1f7035ffb4dd', 'ed31d490-cc0b-59b1-a18f-107e60a46bbf', '804f5e1e-0bef-5f7d-a8eb-bfccc28b4081', 2, '3.99', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('9cea2fe8-ff58-5f73-8eb2-77b932a9eb2c', 'ed31d490-cc0b-59b1-a18f-107e60a46bbf', '1003b6e0-16ae-5290-b5f0-2e9d3b803183', 3, '2.19', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('384b496a-1432-5498-ae21-e25a215eda2c', 'ed31d490-cc0b-59b1-a18f-107e60a46bbf', 'ce05b82c-1ad4-5605-838b-a1b487c3c00a', 1, '2.65', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('cecb5ef2-23ec-563c-8c13-19bf4afa0842', '399cd553-4be5-555b-8d4b-03e71115e8f5', '70703d26-9ed0-5618-9bb2-2e7a116707b3', 2, '2.65', NOW() - INTERVAL '516 days', NOW() - INTERVAL '516 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('743f1724-7364-541e-991c-e53b1a4de89b', 'fb2670e2-434e-5b44-b5ec-913cf1c00b39', 'c7f045b8-ff80-518c-82e5-1295912077d8', 3, '5.32', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('4fd29809-1d7d-52b6-8e68-008caea53ba6', 'fb2670e2-434e-5b44-b5ec-913cf1c00b39', '8c9121d9-045c-5b7e-9b38-49b3b7e35142', 1, '10.53', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('6896e25f-7dc3-5c0c-9a4e-ff57fc26aeb7', 'fb2670e2-434e-5b44-b5ec-913cf1c00b39', '44a7f54f-dc58-4a22-8bbe-178fb68a7713', 2, '10.00', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('5092df9c-8afa-5595-8165-4f7072c5418a', 'fb2670e2-434e-5b44-b5ec-913cf1c00b39', '842f58d4-b287-5a87-86a6-8518a4cfe215', 3, '1058.67', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('591d50dd-e2f3-5580-a918-84de4360d72c', 'be59f341-2d5b-5973-9672-1eb359ad3583', '6d60a1ca-4b0f-56e5-9a88-25f8ecc52d64', 1, '2.19', NOW() - INTERVAL '196 days', NOW() - INTERVAL '196 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a2180629-5eb0-549d-a8fd-d9c7968ef3f2', '8b91040a-ac38-5ac0-a495-ed42e69db228', 'e8a5fd70-82bb-5912-bad4-a73404ba5795', 2, '3.99', NOW() - INTERVAL '197 days', NOW() - INTERVAL '197 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('51619411-704a-57e0-8097-8c4dfed239af', '8b91040a-ac38-5ac0-a495-ed42e69db228', 'cd7f72a1-6989-451d-8d1c-85c8221b29ec', 3, '100.00', NOW() - INTERVAL '197 days', NOW() - INTERVAL '197 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('61e0ae44-1da8-5cdc-a218-660c58701408', '0842e14a-1845-55cc-ad4e-207152e599c1', 'bd6dbba6-bd75-579e-a747-f9c91bfb5a16', 3, '8.53', NOW() - INTERVAL '198 days', NOW() - INTERVAL '198 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3e349faa-a30a-5254-9476-108259d81994', '0842e14a-1845-55cc-ad4e-207152e599c1', '11ffb83b-a5ad-4c12-b563-4048109870bd', 1, '200.00', NOW() - INTERVAL '198 days', NOW() - INTERVAL '198 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('7c1bc01d-e8c1-5064-a55e-9f4fcae8e90c', '0842e14a-1845-55cc-ad4e-207152e599c1', 'c4e15886-fc5b-5c48-bcf4-6837f1bf7249', 2, '1098.67', NOW() - INTERVAL '198 days', NOW() - INTERVAL '198 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('2efa6e48-757f-520f-af02-d5d1e0400ac4', 'de0c2e36-2f58-5db8-8d94-ad22cf098dda', '54dfc1f2-2567-5a3c-a3ab-9631d1632718', 1, '13.20', NOW() - INTERVAL '36 days', NOW() - INTERVAL '36 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('0062220b-c53d-5343-88be-9aeeccbff410', 'd28b67d4-4510-5b95-90ac-6edb27c6ab13', 'b8a486a4-bda3-5db6-b918-fff60aafffa8', 2, '27.87', NOW() - INTERVAL '37 days', NOW() - INTERVAL '37 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('2f0deffe-549e-51c0-ab1d-17b7c73d0d7b', 'd28b67d4-4510-5b95-90ac-6edb27c6ab13', 'f4fbf747-9863-5d29-8e8e-0f895d0e3045', 3, '2.99', NOW() - INTERVAL '37 days', NOW() - INTERVAL '37 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('a6d4808f-5aae-56f6-bff2-38522f012bf1', 'fc1d31a1-5dfe-508f-8c8a-5f8fa220fec0', 'e2e9d241-1cd8-5606-963a-071cf9c4e44b', 3, '29.87', NOW() - INTERVAL '38 days', NOW() - INTERVAL '38 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('3b754db9-0f31-5030-9b02-c8b1fa5e7e59', 'fc1d31a1-5dfe-508f-8c8a-5f8fa220fec0', '51a251a4-7add-5ae6-bd69-a9abbb5aa463', 1, '10.53', NOW() - INTERVAL '38 days', NOW() - INTERVAL '38 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('87caa474-f558-5014-8271-b60a2a0d54f4', 'fc1d31a1-5dfe-508f-8c8a-5f8fa220fec0', '504ed1f1-b7a8-5698-9a96-160a2a0a2514', 2, '3.99', NOW() - INTERVAL '38 days', NOW() - INTERVAL '38 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('be4b4879-ba6c-553b-bc34-4ec3eb15f9b4', 'f5ae77cb-d5f3-52fe-b8ef-49cb744b58bb', '1063c857-124e-51da-b2cd-3b453adc705a', 1, '13.20', NOW() - INTERVAL '39 days', NOW() - INTERVAL '39 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('34d482b6-2862-5a91-821e-f058bb75746c', 'f5ae77cb-d5f3-52fe-b8ef-49cb744b58bb', 'a76f37aa-b93b-5d36-a38a-85063a47b770', 2, '6.53', NOW() - INTERVAL '39 days', NOW() - INTERVAL '39 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('8566fd57-9cf3-5bea-8115-34ca2cc11d4f', 'f5ae77cb-d5f3-52fe-b8ef-49cb744b58bb', 'c059d3c6-4db5-5689-8387-308bd7bbb977', 3, '6.65', NOW() - INTERVAL '39 days', NOW() - INTERVAL '39 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('7bd74101-b5a9-5255-aa07-fe5e07eaa730', 'f5ae77cb-d5f3-52fe-b8ef-49cb744b58bb', '9b158101-ef56-58ca-b83f-462714acff60', 1, '1.32', NOW() - INTERVAL '39 days', NOW() - INTERVAL '39 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('9d6fe662-692f-5c1f-ac11-83d9a417a90d', 'a8704692-2a97-5cc3-8c65-e8a5f8d597ba', '575fdd08-84ca-51ab-821d-2da59344ef3d', 2, '11.00', NOW() - INTERVAL '40 days', NOW() - INTERVAL '40 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('e81cd440-d495-5691-a6bb-97c348526bd0', 'e3be21eb-37ba-5f74-bcb6-2979a05e2b00', '2acc232e-a8b9-5f16-b8c8-7f84ce87ec6b', 1, '4.19', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('4284c0a0-e20b-5ac0-9ce2-de7a337b92bf', 'e3be21eb-37ba-5f74-bcb6-2979a05e2b00', 'c43f094e-bfdb-477b-a718-2a56de358735', 2, '10.00', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('5363d830-2a2f-5780-8e87-e05da9ff88d7', 'e3be21eb-37ba-5f74-bcb6-2979a05e2b00', '01497107-ae60-5ba3-9052-adc90c1a4997', 3, '102.67', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('d3ecce82-d0c8-5179-8382-62ff8bd5f69a', 'e3be21eb-37ba-5f74-bcb6-2979a05e2b00', 'dfff276a-cc12-5125-a52b-2ab4218c8c24', 1, '278.67', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('bef5315c-88d8-5f44-99ef-bb3e3e927fb3', '542e9372-7e51-5d18-9224-b9a6747ea6f1', '0dbfdd53-bb40-54ba-95f9-39aa5adc2cbe', 2, '14.67', NOW() - INTERVAL '216 days', NOW() - INTERVAL '216 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('626f5fc5-66de-5ebb-97c8-e1d5ba4d5804', '34b6258d-1da7-5c56-8b66-5b534ef4c84e', 'cf50b9ea-37ae-5a16-b610-22ae0d214e1f', 3, '8.00', NOW() - INTERVAL '217 days', NOW() - INTERVAL '217 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('41809f99-e84b-5374-b516-694e4f14d274', '34b6258d-1da7-5c56-8b66-5b534ef4c84e', '8917a35a-b143-5a2e-95d8-85f8f9adf7e3', 1, '10.00', NOW() - INTERVAL '217 days', NOW() - INTERVAL '217 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('cb555c31-b92d-5e73-acaf-1f577678fd1c', '6865d985-a9f1-5c1e-9c79-9902ca06ecb0', 'f79c5806-5751-489a-a26e-d5f75624361a', 1, '20.00', NOW() - INTERVAL '218 days', NOW() - INTERVAL '218 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('608e35e1-d33b-5a02-a481-860641490516', '6865d985-a9f1-5c1e-9c79-9902ca06ecb0', 'a818502b-0699-5e17-bf4c-84e9483349a7', 2, '66.60', NOW() - INTERVAL '218 days', NOW() - INTERVAL '218 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('082941ec-4ea3-5dc7-af7f-68302ea05856', '6865d985-a9f1-5c1e-9c79-9902ca06ecb0', '8d3158c5-9318-5b9c-8ddf-b0f8c88b8e18', 3, '95.33', NOW() - INTERVAL '218 days', NOW() - INTERVAL '218 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('218afe58-4f41-552d-a877-dc3974bba73d', 'e409b934-692b-5046-93b1-2174d3267a22', '8ec36fad-1795-49f0-b521-9396226da5e8', 2, '33.33', NOW() - INTERVAL '219 days', NOW() - INTERVAL '219 days') ON CONFLICT (id) DO NOTHING;
INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) VALUES
('bd2e3114-bd09-5c2d-9540-510778b4a020', 'e409b934-692b-5046-93b1-2174d3267a22', '00bebff6-1918-5340-9d37-3e3897b679f3', 3, '26.60', NOW() - INTERVAL '219 days', NOW() - INTERVAL '219 days') ON CONFLICT (id) DO NOTHING;
