ALTER TABLE public_moderation_actions
  DROP CONSTRAINT public_moderation_actions_actor_account_id_fkey;

ALTER TABLE public_moderation_actions
  ALTER COLUMN actor_account_id DROP NOT NULL;

ALTER TABLE public_moderation_actions
  ADD CONSTRAINT public_moderation_actions_actor_account_id_fkey
  FOREIGN KEY (actor_account_id) REFERENCES accounts(id) ON DELETE SET NULL;

COMMENT ON COLUMN public_moderation_actions.actor_account_id IS
  'Account that took the action when still retained; actions survive later account erasure.';
