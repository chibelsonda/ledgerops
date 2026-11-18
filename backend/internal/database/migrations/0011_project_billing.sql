CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id),
    project_code TEXT NOT NULL CHECK (char_length(project_code) BETWEEN 2 AND 30),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 150),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'on_hold', 'completed', 'cancelled')),
    currency_code CHAR(3) NOT NULL,
    start_on DATE NOT NULL,
    end_on DATE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT projects_dates CHECK (end_on IS NULL OR end_on >= start_on),
    UNIQUE (organization_id, project_code)
);

CREATE INDEX projects_org_status_idx ON projects (organization_id, status, created_at DESC);

CREATE TABLE project_contracts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id),
    contract_number TEXT NOT NULL CHECK (char_length(contract_number) BETWEEN 2 AND 40),
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 2 AND 150),
    original_value_minor BIGINT NOT NULL CHECK (original_value_minor > 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_by UUID NOT NULL REFERENCES users(id),
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT CHECK (review_note IS NULL OR char_length(review_note) BETWEEN 3 AND 500),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT project_contract_review_state CHECK (
        (status = 'pending' AND reviewed_by IS NULL AND reviewed_at IS NULL AND review_note IS NULL)
        OR (status IN ('approved', 'rejected') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND review_note IS NOT NULL)
    ),
    UNIQUE (project_id),
    UNIQUE (organization_id, contract_number)
);

CREATE INDEX project_contracts_org_status_idx ON project_contracts (organization_id, status, requested_at DESC);

CREATE TABLE contract_change_orders (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES project_contracts(id),
    change_order_number TEXT NOT NULL CHECK (char_length(change_order_number) BETWEEN 2 AND 40),
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 2 AND 150),
    description TEXT NOT NULL CHECK (char_length(description) BETWEEN 3 AND 1000),
    amount_minor BIGINT NOT NULL CHECK (amount_minor <> 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_by UUID NOT NULL REFERENCES users(id),
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT CHECK (review_note IS NULL OR char_length(review_note) BETWEEN 3 AND 500),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT change_order_review_state CHECK (
        (status = 'pending' AND reviewed_by IS NULL AND reviewed_at IS NULL AND review_note IS NULL)
        OR (status IN ('approved', 'rejected') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND review_note IS NOT NULL)
    ),
    UNIQUE (contract_id, change_order_number)
);

CREATE INDEX contract_change_orders_contract_idx ON contract_change_orders (contract_id, requested_at DESC);

CREATE TABLE billing_milestones (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES project_contracts(id),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 150),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    due_on DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'invoiced')),
    invoice_id UUID UNIQUE REFERENCES invoices(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    invoiced_at TIMESTAMPTZ,
    CONSTRAINT billing_milestone_invoice_state CHECK (
        (status = 'planned' AND invoice_id IS NULL AND invoiced_at IS NULL)
        OR (status = 'invoiced' AND invoice_id IS NOT NULL AND invoiced_at IS NOT NULL)
    )
);

CREATE INDEX billing_milestones_contract_idx ON billing_milestones (contract_id, due_on, created_at);

CREATE FUNCTION protect_project_contract()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'project contracts cannot be deleted' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'pending' THEN
        RAISE EXCEPTION 'finalized project contracts are immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
       OR NEW.project_id IS DISTINCT FROM OLD.project_id
       OR NEW.contract_number IS DISTINCT FROM OLD.contract_number
       OR NEW.title IS DISTINCT FROM OLD.title
       OR NEW.original_value_minor IS DISTINCT FROM OLD.original_value_minor
       OR NEW.requested_by IS DISTINCT FROM OLD.requested_by
       OR NEW.requested_at IS DISTINCT FROM OLD.requested_at
       OR NEW.status NOT IN ('approved', 'rejected') THEN
        RAISE EXCEPTION 'only a pending contract decision is allowed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER project_contracts_are_immutable
    BEFORE UPDATE OR DELETE ON project_contracts
    FOR EACH ROW EXECUTE FUNCTION protect_project_contract();

CREATE FUNCTION protect_change_order()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'change orders cannot be deleted' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'pending' THEN
        RAISE EXCEPTION 'finalized change orders are immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
       OR NEW.contract_id IS DISTINCT FROM OLD.contract_id
       OR NEW.change_order_number IS DISTINCT FROM OLD.change_order_number
       OR NEW.title IS DISTINCT FROM OLD.title
       OR NEW.description IS DISTINCT FROM OLD.description
       OR NEW.amount_minor IS DISTINCT FROM OLD.amount_minor
       OR NEW.requested_by IS DISTINCT FROM OLD.requested_by
       OR NEW.requested_at IS DISTINCT FROM OLD.requested_at
       OR NEW.status NOT IN ('approved', 'rejected') THEN
        RAISE EXCEPTION 'only a pending change-order decision is allowed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER contract_change_orders_are_immutable
    BEFORE UPDATE OR DELETE ON contract_change_orders
    FOR EACH ROW EXECUTE FUNCTION protect_change_order();

CREATE FUNCTION protect_billing_milestone()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'billing milestones cannot be deleted' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'planned' THEN
        RAISE EXCEPTION 'invoiced billing milestones are immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
       OR NEW.contract_id IS DISTINCT FROM OLD.contract_id
       OR NEW.name IS DISTINCT FROM OLD.name
       OR NEW.amount_minor IS DISTINCT FROM OLD.amount_minor
       OR NEW.due_on IS DISTINCT FROM OLD.due_on
       OR NEW.created_by IS DISTINCT FROM OLD.created_by
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.status <> 'invoiced' THEN
        RAISE EXCEPTION 'only invoicing a planned milestone is allowed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER billing_milestones_are_immutable
    BEFORE UPDATE OR DELETE ON billing_milestones
    FOR EACH ROW EXECUTE FUNCTION protect_billing_milestone();
