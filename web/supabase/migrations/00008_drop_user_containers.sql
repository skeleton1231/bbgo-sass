-- Drop user_containers table — strategy configs now live on disk (StrategyStore)
-- and container status is derived from Docker + YAML file existence.
drop table if exists public.user_containers;
