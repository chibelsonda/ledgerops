INSERT INTO organizations (id, name, base_currency)
VALUES ('11111111-1111-4111-8111-111111111111', 'Northstar Creative Studio', 'USD')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, display_name, password_hash) VALUES
('81111111-1111-7111-8111-111111111111', 'admin@ledgerops.demo', 'Alex Morgan', '$argon2id$v=19$m=19456,t=2,p=1$Qumcjm0Qm1+fhTQNdfDEsQ$SmS0joQ7H8aDeEyOkjqp8en4Zbjh3+F/c7z+FZAgJUw'),
('82222222-2222-7222-8222-222222222222', 'mara@ledgerops.demo', 'Mara Chen', '$argon2id$v=19$m=19456,t=2,p=1$VCi7O8opSRzpO0TyXCmhJQ$Yhi8/qusXsFHj5qN03uj3zxepw/M08NON1GYet2+J0U'),
('83333333-3333-7333-8333-333333333333', 'reviewer@ledgerops.demo', 'Riley Shah', '$argon2id$v=19$m=19456,t=2,p=1$LCWYS46rRIGP32xYFkrEgg$vs23QzQgGNckwQENpmCyIrEIxHO2aGYXB9scS3bwB2Y')
ON CONFLICT (id) DO NOTHING;

INSERT INTO organization_memberships (organization_id, user_id, role) VALUES
('11111111-1111-4111-8111-111111111111', '81111111-1111-7111-8111-111111111111', 'admin'),
('11111111-1111-4111-8111-111111111111', '82222222-2222-7222-8222-222222222222', 'controller'),
('11111111-1111-4111-8111-111111111111', '83333333-3333-7333-8333-333333333333', 'reviewer')
ON CONFLICT (organization_id, user_id) DO NOTHING;

INSERT INTO close_periods (id, organization_id, month_start, due_on, status, created_by) VALUES
('a1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', date_trunc('month', CURRENT_DATE)::date, (date_trunc('month', CURRENT_DATE) + interval '1 month' + interval '5 days')::date, 'draft', '82222222-2222-7222-8222-222222222222')
ON CONFLICT (id) DO NOTHING;

INSERT INTO close_tasks (id, organization_id, close_period_id, title, category, position, status, evidence_note, completed_by, completed_at) VALUES
('a2111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Reconcile operating bank accounts', 'Cash', 1, 'completed', 'Matched the operating account through the final business day.', '82222222-2222-7222-8222-222222222222', now() - interval '2 hours'),
('a2222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Review receivables and credit notes', 'Revenue', 2, 'completed', 'Reviewed aging, credits, and payment applications.', '82222222-2222-7222-8222-222222222222', now() - interval '90 minutes'),
('a2333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Record accruals and vendor obligations', 'Expenses', 3, 'pending', NULL, NULL, NULL),
('a2444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Verify payroll and operating expenses', 'Expenses', 4, 'pending', NULL, NULL, NULL),
('a2555555-5555-4555-8555-555555555555', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Confirm project billing completeness', 'Projects', 5, 'pending', NULL, NULL, NULL),
('a2666666-6666-4666-8666-666666666666', '11111111-1111-4111-8111-111111111111', 'a1111111-1111-4111-8111-111111111111', 'Prepare management variance review', 'Reporting', 6, 'pending', NULL, NULL, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO cash_flow_scenarios (id, organization_id, name, collection_rate_bps, collection_delay_days, expense_growth_bps, milestone_probability_bps, created_by) VALUES
('b1111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'Base case', 8500, 7, 0, 7000, '82222222-2222-7222-8222-222222222222'),
('b1222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'Optimistic', 9500, 0, -500, 9000, '82222222-2222-7222-8222-222222222222'),
('b1333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', 'Conservative', 7000, 21, 1000, 5000, '82222222-2222-7222-8222-222222222222')
ON CONFLICT (id) DO NOTHING;

INSERT INTO cash_flow_adjustments (id, organization_id, scenario_id, week_start, direction, label, amount_minor, created_by) VALUES
('b2111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'b1111111-1111-4111-8111-111111111111', date_trunc('week', CURRENT_DATE)::date + 28, 'outflow', 'Annual insurance renewal', 180000, '82222222-2222-7222-8222-222222222222')
ON CONFLICT (id) DO NOTHING;

INSERT INTO clients (id, organization_id, external_id, name, email) VALUES
('21111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'qbo-customer-101', 'Arc & Co.', 'finance@arc.example'),
('22222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'qbo-customer-102', 'Fieldwork Labs', 'ops@fieldwork.example'),
('23333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', 'qbo-customer-103', 'Juniper Market', 'ap@juniper.example'),
('24444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', 'qbo-customer-104', 'Waypoint Health', 'billing@waypoint.example')
ON CONFLICT (id) DO NOTHING;

INSERT INTO invoices (id, organization_id, client_id, external_id, invoice_number, status, currency_code, amount_minor, issued_on, due_on, source, source_updated_at) VALUES
('31111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '21111111-1111-4111-8111-111111111111', 'qbo-invoice-501', 'INV-2026-051', 'paid',    'USD', 1285000, CURRENT_DATE - 150, CURRENT_DATE - 120, 'quickbooks', now() - interval '2 hours'),
('32222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'qbo-invoice-502', 'INV-2026-052', 'paid',    'USD',  940000, CURRENT_DATE - 118, CURRENT_DATE -  88, 'quickbooks', now() - interval '2 hours'),
('33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '23333333-3333-4333-8333-333333333333', 'qbo-invoice-503', 'INV-2026-053', 'paid',    'USD', 1475000, CURRENT_DATE -  88, CURRENT_DATE -  58, 'quickbooks', now() - interval '2 hours'),
('34444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', '24444444-4444-4444-8444-444444444444', 'qbo-invoice-504', 'INV-2026-054', 'paid',    'USD', 1660000, CURRENT_DATE -  58, CURRENT_DATE -  28, 'quickbooks', now() - interval '2 hours'),
('35555555-5555-4555-8555-555555555555', '11111111-1111-4111-8111-111111111111', '21111111-1111-4111-8111-111111111111', 'qbo-invoice-505', 'INV-2026-055', 'overdue', 'USD',  825000, CURRENT_DATE -  45, CURRENT_DATE -  15, 'quickbooks', now() - interval '2 hours'),
('36666666-6666-4666-8666-666666666666', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'qbo-invoice-506', 'INV-2026-056', 'partial', 'USD', 1190000, CURRENT_DATE -  32, CURRENT_DATE -   2, 'quickbooks', now() - interval '2 hours'),
('37777777-7777-4777-8777-777777777777', '11111111-1111-4111-8111-111111111111', '23333333-3333-4333-8333-333333333333', 'qbo-invoice-507', 'INV-2026-057', 'open',    'USD', 1540000, CURRENT_DATE -  14, CURRENT_DATE +  16, 'quickbooks', now() - interval '2 hours'),
('38888888-8888-4888-8888-888888888888', '11111111-1111-4111-8111-111111111111', '24444444-4444-4444-8444-444444444444', 'qbo-invoice-508', 'INV-2026-058', 'open',    'USD',  980000, CURRENT_DATE -   5, CURRENT_DATE +  25, 'quickbooks', now() - interval '2 hours')
ON CONFLICT (id) DO NOTHING;

INSERT INTO payments (id, organization_id, invoice_id, external_id, currency_code, amount_minor, received_on, source) VALUES
('41111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '31111111-1111-4111-8111-111111111111', 'qbo-payment-701', 'USD', 1285000, CURRENT_DATE - 124, 'quickbooks'),
('42222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', '32222222-2222-4222-8222-222222222222', 'qbo-payment-702', 'USD',  940000, CURRENT_DATE -  91, 'quickbooks'),
('43333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '33333333-3333-4333-8333-333333333333', 'qbo-payment-703', 'USD', 1475000, CURRENT_DATE -  61, 'quickbooks'),
('44444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', '34444444-4444-4444-8444-444444444444', 'qbo-payment-704', 'USD', 1660000, CURRENT_DATE -  30, 'quickbooks'),
('45555555-5555-4555-8555-555555555555', '11111111-1111-4111-8111-111111111111', '36666666-6666-4666-8666-666666666666', 'qbo-payment-705', 'USD',  400000, CURRENT_DATE -  12, 'quickbooks')
ON CONFLICT (id) DO NOTHING;

INSERT INTO expenses (id, organization_id, external_id, category, vendor_name, currency_code, amount_minor, incurred_on, source) VALUES
('51111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'qbo-expense-801', 'Payroll', 'Gusto', 'USD', 920000, CURRENT_DATE - 138, 'quickbooks'),
('52222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', 'qbo-expense-802', 'Software', 'Cloud services', 'USD', 185000, CURRENT_DATE - 105, 'quickbooks'),
('53333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', 'qbo-expense-803', 'Payroll', 'Gusto', 'USD', 975000, CURRENT_DATE -  75, 'quickbooks'),
('54444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', 'qbo-expense-804', 'Contractors', 'Specialist network', 'USD', 430000, CURRENT_DATE -  43, 'quickbooks'),
('55555555-5555-4555-8555-555555555555', '11111111-1111-4111-8111-111111111111', 'qbo-expense-805', 'Payroll', 'Gusto', 'USD', 1010000, CURRENT_DATE -  17, 'quickbooks'),
('56666666-6666-4666-8666-666666666666', '11111111-1111-4111-8111-111111111111', 'qbo-expense-806', 'Software', 'Cloud services', 'USD', 212000, CURRENT_DATE -   4, 'quickbooks')
ON CONFLICT (id) DO NOTHING;

INSERT INTO budgets (organization_id, month_start, category, amount_minor) VALUES
('11111111-1111-4111-8111-111111111111', date_trunc('month', CURRENT_DATE)::date, 'Payroll', 1050000),
('11111111-1111-4111-8111-111111111111', date_trunc('month', CURRENT_DATE)::date, 'Software', 250000),
('11111111-1111-4111-8111-111111111111', date_trunc('month', CURRENT_DATE)::date, 'Contractors', 500000),
('11111111-1111-4111-8111-111111111111', date_trunc('month', CURRENT_DATE)::date, 'Marketing', 175000)
ON CONFLICT (organization_id, month_start, category) DO NOTHING;

INSERT INTO sync_runs (id, organization_id, provider, status, records_processed, started_at, finished_at, created_at) VALUES
('61111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'quickbooks', 'succeeded', 37, now() - interval '2 hours 1 minute', now() - interval '2 hours', now() - interval '2 hours 1 minute')
ON CONFLICT (id) DO NOTHING;

INSERT INTO reconciliation_items (id, organization_id, invoice_id, kind, severity, title, description, amount_minor, currency_code, status, created_at) VALUES
('71111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '35555555-5555-4555-8555-555555555555', 'overdue_invoice', 'high', 'Invoice 15 days overdue', 'Arc & Co. has not paid INV-2026-055. Confirm collection status or record an offline payment.', 825000, 'USD', 'open', now() - interval '3 hours'),
('72222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', '36666666-6666-4666-8666-666666666666', 'partial_payment', 'medium', 'Partial payment needs review', 'INV-2026-056 has an unapplied balance of $7,900.00 after the latest payment.', 790000, 'USD', 'open', now() - interval '2 hours'),
('73333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', NULL, 'budget_variance', 'medium', 'Software spend approaching budget', 'Current software spend is at 85% of the monthly budget with 12 days remaining.', 212000, 'USD', 'open', now() - interval '1 hour')
ON CONFLICT (id) DO NOTHING;

INSERT INTO projects (id, organization_id, client_id, project_code, name, status, currency_code, start_on, end_on, created_by, created_at) VALUES
('91111111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '21111111-1111-4111-8111-111111111111', 'ARC-PORTAL', 'Arc client portal modernization', 'active', 'USD', CURRENT_DATE - 45, CURRENT_DATE + 120, '82222222-2222-7222-8222-222222222222', now() - interval '45 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO project_contracts (id, organization_id, project_id, contract_number, title, original_value_minor, status, requested_by, reviewed_by, review_note, requested_at, reviewed_at) VALUES
('92222222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', '91111111-1111-4111-8111-111111111111', 'CTR-ARC-2026', 'Portal design and delivery', 8000000, 'approved', '82222222-2222-7222-8222-222222222222', '81111111-1111-7111-8111-111111111111', 'Approved against the signed client agreement.', now() - interval '45 days', now() - interval '44 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO contract_change_orders (id, organization_id, contract_id, change_order_number, title, description, amount_minor, status, requested_by, reviewed_by, review_note, requested_at, reviewed_at) VALUES
('93333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '92222222-2222-4222-8222-222222222222', 'CO-ARC-001', 'Analytics workspace', 'Adds the client analytics workspace and export controls.', 1200000, 'approved', '82222222-2222-7222-8222-222222222222', '81111111-1111-7111-8111-111111111111', 'Approved after client sign-off.', now() - interval '30 days', now() - interval '29 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO billing_milestones (id, organization_id, contract_id, name, amount_minor, due_on, status, created_by, created_at) VALUES
('94444444-4444-4444-8444-444444444444', '11111111-1111-4111-8111-111111111111', '92222222-2222-4222-8222-222222222222', 'Discovery and architecture', 3000000, CURRENT_DATE + 5, 'planned', '82222222-2222-7222-8222-222222222222', now() - interval '28 days'),
('95555555-5555-4555-8555-555555555555', '11111111-1111-4111-8111-111111111111', '92222222-2222-4222-8222-222222222222', 'Application delivery', 2500000, CURRENT_DATE + 60, 'planned', '82222222-2222-7222-8222-222222222222', now() - interval '28 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO audit_events (id, organization_id, actor, action, entity_type, entity_id, metadata, created_at) VALUES
('96666666-6666-4666-8666-666666666666', '11111111-1111-4111-8111-111111111111', 'mara@ledgerops.demo', 'project.created', 'project', '91111111-1111-4111-8111-111111111111', '{"projectCode":"ARC-PORTAL","contractNumber":"CTR-ARC-2026","originalValueMinor":8000000}', now() - interval '45 days'),
('97777777-7777-4777-8777-777777777777', '11111111-1111-4111-8111-111111111111', 'admin@ledgerops.demo', 'contract.approved', 'project', '91111111-1111-4111-8111-111111111111', '{"note":"Approved against the signed client agreement."}', now() - interval '44 days'),
('98888888-8888-4888-8888-888888888888', '11111111-1111-4111-8111-111111111111', 'admin@ledgerops.demo', 'change_order.approved', 'project', '91111111-1111-4111-8111-111111111111', '{"number":"CO-ARC-001","amountMinor":1200000}', now() - interval '29 days')
ON CONFLICT (id) DO NOTHING;
