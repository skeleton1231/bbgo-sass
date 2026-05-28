-- Add dual-container support: live/paper mode per user
alter table public.user_containers
  add column if not exists mode text not null default 'live' check (mode in ('live', 'paper'));

-- Change PK from user_id alone to (user_id, mode)
alter table public.user_containers drop constraint user_containers_pkey;
alter table public.user_containers add primary key (user_id, mode);

-- Add 'starting' to allowed status values
alter table public.user_containers drop constraint user_containers_status_check;
alter table public.user_containers add constraint user_containers_status_check
  check (status in ('running', 'stopped', 'error', 'starting'));
