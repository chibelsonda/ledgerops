CREATE OR REPLACE FUNCTION prevent_payment_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'payment records are immutable' USING ERRCODE = '55000';
    END IF;
    IF OLD.external_id LIKE 'quickbooks:payment:%'
       AND NEW.source_deleted_at IS DISTINCT FROM OLD.source_deleted_at
       AND (to_jsonb(NEW) - 'source_deleted_at') = (to_jsonb(OLD) - 'source_deleted_at') THEN
        RETURN NEW;
    END IF;
    IF OLD.source = 'manual'
       AND OLD.external_id IS NULL
       AND NEW.source = OLD.source
       AND NEW.external_id IS NOT NULL
       AND (to_jsonb(NEW) - 'external_id') = (to_jsonb(OLD) - 'external_id') THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'payment records are immutable' USING ERRCODE = '55000';
END;
$$;
