CREATE TABLE cash_flow_scenarios (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    collection_rate_bps INTEGER NOT NULL CHECK (collection_rate_bps BETWEEN 0 AND 10000),
    collection_delay_days INTEGER NOT NULL CHECK (collection_delay_days BETWEEN 0 AND 180),
    expense_growth_bps INTEGER NOT NULL CHECK (expense_growth_bps BETWEEN -10000 AND 50000),
    milestone_probability_bps INTEGER NOT NULL CHECK (milestone_probability_bps BETWEEN 0 AND 10000),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);

CREATE INDEX cash_flow_scenarios_org_created_idx ON cash_flow_scenarios (organization_id, created_at);

CREATE TABLE cash_flow_adjustments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    scenario_id UUID NOT NULL REFERENCES cash_flow_scenarios(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('inflow', 'outflow')),
    label TEXT NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (week_start = date_trunc('week', week_start)::date)
);

CREATE INDEX cash_flow_adjustments_scenario_week_idx ON cash_flow_adjustments (scenario_id, week_start);

CREATE OR REPLACE FUNCTION prevent_forecast_adjustment_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'forecast adjustments are append-only' USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER cash_flow_adjustment_update_guard BEFORE UPDATE OR DELETE ON cash_flow_adjustments
FOR EACH ROW EXECUTE FUNCTION prevent_forecast_adjustment_mutation();
