-- Seed data for local count API testing (usergroup_db)
-- Matches current group expression example:
--   $.kycStatus == "PASSED" && $.totalFiatDepositUSD >= 200
-- Expected matched: user_id 1001, 1002, 1005 (3)

INSERT INTO user_full_info (user_id, profile) VALUES
(1001, '{
  "registeredAt": 1700000000,
  "market": "SG",
  "kycStatus": "PASSED",
  "totalFiatDepositUSD": 500,
  "fiatDepositCount": 3,
  "totalPurchaseAmountUSD": 1200,
  "purchaseCount": 4,
  "isRiskUser": false
}'::jsonb),
(1002, '{
  "registeredAt": 1701000000,
  "market": "HK",
  "kycStatus": "PASSED",
  "totalFiatDepositUSD": 200,
  "fiatDepositCount": 1,
  "totalPurchaseAmountUSD": 200,
  "purchaseCount": 1,
  "isRiskUser": false
}'::jsonb),
(1003, '{
  "registeredAt": 1702000000,
  "market": "SG",
  "kycStatus": "PASSED",
  "totalFiatDepositUSD": 199.99,
  "fiatDepositCount": 2,
  "totalPurchaseAmountUSD": 50,
  "purchaseCount": 1,
  "isRiskUser": false
}'::jsonb),
(1004, '{
  "registeredAt": 1703000000,
  "market": "US",
  "kycStatus": "PENDING",
  "totalFiatDepositUSD": 1000,
  "fiatDepositCount": 5,
  "totalPurchaseAmountUSD": 800,
  "purchaseCount": 2,
  "isRiskUser": false
}'::jsonb),
(1005, '{
  "registeredAt": 1704000000,
  "market": "SG",
  "kycStatus": "PASSED",
  "totalFiatDepositUSD": 3000,
  "fiatDepositCount": 10,
  "totalPurchaseAmountUSD": 5000,
  "purchaseCount": 8,
  "isRiskUser": true
}'::jsonb),
(1006, '{
  "registeredAt": 1705000000,
  "market": "HK",
  "kycStatus": "REJECTED",
  "totalFiatDepositUSD": 0,
  "fiatDepositCount": 0,
  "totalPurchaseAmountUSD": 0,
  "purchaseCount": 0,
  "isRiskUser": true
}'::jsonb)
ON CONFLICT (user_id) DO UPDATE SET
  profile = EXCLUDED.profile,
  updated_at = NOW();
