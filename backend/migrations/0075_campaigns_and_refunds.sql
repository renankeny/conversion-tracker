-- 0075_campaigns_and_refunds.sql
-- Motor nativo de Reembolsos/Chargebacks (sale_refunds), custos financeiros por Pixel
-- (Lucro Líquido Real em /campanhas), status de estorno no CRM e alerta sale_refunded.

CREATE TABLE IF NOT EXISTS sale_refunds (
    id                       BIGSERIAL PRIMARY KEY,
    account_id               BIGINT NOT NULL DEFAULT 1 REFERENCES accounts(id) ON DELETE CASCADE,
    pixel_name               TEXT NOT NULL DEFAULT '',
    gateway                  TEXT NOT NULL DEFAULT '',
    order_id                 TEXT NOT NULL,
    refund_type              VARCHAR(20) NOT NULL DEFAULT 'refund', -- 'refund' | 'chargeback'
    amount                   NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency                 VARCHAR(10) NOT NULL DEFAULT 'BRL',
    original_event_id        TEXT NOT NULL DEFAULT '',
    tracked_user_id          BIGINT REFERENCES tracked_users(id) ON DELETE SET NULL,
    group_id                 BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    crm_lead_id              BIGINT REFERENCES crm_leads(id) ON DELETE SET NULL,
    product_name             TEXT NOT NULL DEFAULT '',
    google_retraction_status VARCHAR(24) NOT NULL DEFAULT 'pending', -- pending | sent | skipped | failed
    google_retraction_error  TEXT NOT NULL DEFAULT '',
    ga4_refund_status        VARCHAR(24) NOT NULL DEFAULT 'pending', -- pending | sent | skipped | failed
    ga4_refund_error         TEXT NOT NULL DEFAULT '',
    raw_payload              JSONB,
    refunded_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sale_refunds_acc_gw_order UNIQUE (account_id, gateway, order_id)
);

CREATE INDEX IF NOT EXISTS idx_sale_refunds_acc_refunded
    ON sale_refunds(account_id, refunded_at DESC);
CREATE INDEX IF NOT EXISTS idx_sale_refunds_acc_pixel_refunded
    ON sale_refunds(account_id, pixel_name, refunded_at DESC);
CREATE INDEX IF NOT EXISTS idx_sale_refunds_tracked_user
    ON sale_refunds(account_id, tracked_user_id)
    WHERE tracked_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sale_refunds_group
    ON sale_refunds(account_id, group_id)
    WHERE group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sale_refunds_order
    ON sale_refunds(account_id, order_id);

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS gateway_fee_pct   NUMERIC(6,3) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gateway_fixed_fee NUMERIC(10,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_pct           NUMERIC(6,3) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS product_cost      NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE crm_leads
    ADD COLUMN IF NOT EXISTS refund_status   VARCHAR(20) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS refunded_amount NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE user_notification_preferences
    ADD COLUMN IF NOT EXISTS sale_refunded BOOLEAN NOT NULL DEFAULT TRUE;
