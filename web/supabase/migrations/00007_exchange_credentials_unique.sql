alter table public.exchange_credentials
  add constraint exchange_credentials_user_exchange_testnet_unique
  unique (user_id, exchange, is_testnet);
