-- Drop existing RESTRICT foreign keys and replace with CASCADE (idempotent)

ALTER TABLE event_counters DROP CONSTRAINT IF EXISTS event_counters_account_id_fkey;
ALTER TABLE event_counters ADD CONSTRAINT event_counters_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE pixels DROP CONSTRAINT IF EXISTS pixels_account_id_fkey;
ALTER TABLE pixels ADD CONSTRAINT pixels_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE sent_events DROP CONSTRAINT IF EXISTS sent_events_account_id_fkey;
ALTER TABLE sent_events ADD CONSTRAINT sent_events_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE tracked_users DROP CONSTRAINT IF EXISTS tracked_users_account_id_fkey;
ALTER TABLE tracked_users ADD CONSTRAINT tracked_users_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE url_mappings DROP CONSTRAINT IF EXISTS url_mappings_account_id_fkey;
ALTER TABLE url_mappings ADD CONSTRAINT url_mappings_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE whatsapp_instances DROP CONSTRAINT IF EXISTS whatsapp_instances_account_id_fkey;
ALTER TABLE whatsapp_instances ADD CONSTRAINT whatsapp_instances_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;
