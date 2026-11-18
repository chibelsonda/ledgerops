CREATE TABLE close_periods (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    month_start DATE NOT NULL,
    due_on DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_review', 'closed')),
    created_by UUID NOT NULL REFERENCES users(id),
    submitted_by UUID REFERENCES users(id),
    submitted_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT,
    reviewed_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (month_start = date_trunc('month', month_start)::date),
    CHECK (due_on >= month_start),
    UNIQUE (organization_id, month_start)
);

CREATE INDEX close_periods_org_month_idx ON close_periods (organization_id, month_start DESC);

CREATE TABLE close_tasks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    close_period_id UUID NOT NULL REFERENCES close_periods(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    category TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position > 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    evidence_note TEXT,
    completed_by UUID REFERENCES users(id),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (close_period_id, position)
);

CREATE INDEX close_tasks_period_position_idx ON close_tasks (close_period_id, position);

CREATE OR REPLACE FUNCTION protect_closed_period() RETURNS trigger AS $$
BEGIN
    IF OLD.status = 'closed' AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'closed accounting periods are immutable' USING ERRCODE = '55000';
    END IF;
    IF OLD.status = 'draft' AND NEW.status NOT IN ('draft', 'in_review') THEN
        RAISE EXCEPTION 'invalid close-period transition' USING ERRCODE = '55000';
    END IF;
    IF OLD.status = 'in_review' AND NEW.status NOT IN ('in_review', 'draft', 'closed') THEN
        RAISE EXCEPTION 'invalid close-period transition' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER close_period_update_guard BEFORE UPDATE ON close_periods
FOR EACH ROW EXECUTE FUNCTION protect_closed_period();

CREATE OR REPLACE FUNCTION protect_close_task() RETURNS trigger AS $$
DECLARE period_status TEXT;
BEGIN
    SELECT status INTO period_status FROM close_periods WHERE id = COALESCE(NEW.close_period_id, OLD.close_period_id);
    IF period_status = 'closed' THEN
        RAISE EXCEPTION 'tasks in a closed accounting period are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER close_task_update_guard BEFORE UPDATE OR DELETE ON close_tasks
FOR EACH ROW EXECUTE FUNCTION protect_close_task();

CREATE OR REPLACE FUNCTION prevent_close_period_delete() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'close periods cannot be deleted' USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER close_period_delete_guard BEFORE DELETE ON close_periods
FOR EACH ROW EXECUTE FUNCTION prevent_close_period_delete();
