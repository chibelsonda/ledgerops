package forecast

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"gorm.io/gorm"
)

type PostgresRepository struct{ db *gorm.DB }

func NewPostgresRepository(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

const scenarioProjection = `SELECT s.id::text AS id,s.name,s.collection_rate_bps,s.collection_delay_days,s.expense_growth_bps,s.milestone_probability_bps,u.email AS created_by,to_char(s.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at FROM cash_flow_scenarios s JOIN users u ON u.id=s.created_by`

func (r *PostgresRepository) ListScenarios(ctx context.Context, organizationID string) ([]Scenario, error) {
	items := make([]Scenario, 0)
	if err := r.db.WithContext(ctx).Raw(scenarioProjection+` WHERE s.organization_id=? ORDER BY s.created_at,s.name`, organizationID).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list cash-flow scenarios: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Get(ctx context.Context, organizationID, scenarioID string) (Forecast, error) {
	var result Forecast
	if err := r.db.WithContext(ctx).Raw(scenarioProjection+` WHERE s.organization_id=? AND s.id=?`, organizationID, scenarioID).Scan(&result.Scenario).Error; err != nil {
		return result, fmt.Errorf("get cash-flow scenario: %w", err)
	}
	if result.Scenario.ID == "" {
		return result, apperrors.ErrNotFound
	}
	var organization struct{ CurrencyCode string }
	if err := r.db.WithContext(ctx).Raw(`SELECT base_currency AS currency_code FROM organizations WHERE id=?`, organizationID).Scan(&organization).Error; err != nil {
		return result, fmt.Errorf("get forecast currency: %w", err)
	}
	result.CurrencyCode = organization.CurrencyCode
	var opening struct{ OpeningCashMinor int64 }
	if err := r.db.WithContext(ctx).Raw(`SELECT ((SELECT COALESCE(SUM(amount_minor),0)::bigint FROM payments WHERE organization_id=@organization_id)-(SELECT COALESCE(SUM(amount_minor),0)::bigint FROM invoice_adjustments WHERE organization_id=@organization_id AND kind='payment_reversal' AND status='approved')-(SELECT COALESCE(SUM(amount_minor),0)::bigint FROM expenses WHERE organization_id=@organization_id))::bigint AS opening_cash_minor`, sql.Named("organization_id", organizationID)).Scan(&opening).Error; err != nil {
		return result, fmt.Errorf("calculate opening cash: %w", err)
	}
	result.OpeningCashMinor = opening.OpeningCashMinor

	type projectedItem struct {
		ExpectedOn, Source string
		AmountMinor        int64
	}
	items := make([]projectedItem, 0)
	if err := r.db.WithContext(ctx).Raw(`
		WITH paid AS (SELECT p.invoice_id,(SUM(p.amount_minor)-COALESCE(MAX(rev.total),0))::bigint AS total FROM payments p LEFT JOIN (SELECT invoice_id,SUM(amount_minor)::bigint AS total FROM invoice_adjustments WHERE organization_id=? AND kind='payment_reversal' AND status='approved' GROUP BY invoice_id) rev ON rev.invoice_id=p.invoice_id WHERE p.organization_id=? GROUP BY p.invoice_id),
		credits AS (SELECT invoice_id,SUM(amount_minor)::bigint AS total FROM invoice_adjustments WHERE organization_id=? AND kind='credit_note' AND status='approved' GROUP BY invoice_id)
		SELECT to_char(GREATEST(i.due_on + (? * interval '1 day'),CURRENT_DATE),'YYYY-MM-DD') AS expected_on,'invoice' AS source,(GREATEST(i.amount_minor-COALESCE(p.total,0)-COALESCE(c.total,0),0)*?/10000)::bigint AS amount_minor
		FROM invoices i LEFT JOIN paid p ON p.invoice_id=i.id LEFT JOIN credits c ON c.invoice_id=i.id
		WHERE i.organization_id=? AND i.status IN ('open','partial','overdue')
		UNION ALL
		SELECT to_char(GREATEST(m.due_on + (? * interval '1 day'),CURRENT_DATE),'YYYY-MM-DD'),'project',(m.amount_minor*?/10000)::bigint
		FROM billing_milestones m JOIN project_contracts pc ON pc.id=m.contract_id
		WHERE m.organization_id=? AND m.status='planned' AND pc.status='approved'`,
		organizationID, organizationID, organizationID,
		result.Scenario.CollectionDelayDays, result.Scenario.CollectionRateBPS, organizationID,
		result.Scenario.CollectionDelayDays, result.Scenario.MilestoneProbabilityBPS, organizationID,
	).Scan(&items).Error; err != nil {
		return result, fmt.Errorf("load projected inflows: %w", err)
	}

	var expenseRunRate struct{ Total int64 }
	if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(SUM(amount_minor),0)::bigint AS total FROM expenses WHERE organization_id=? AND incurred_on>=CURRENT_DATE-91`, organizationID).Scan(&expenseRunRate).Error; err != nil {
		return result, fmt.Errorf("load operating run-rate: %w", err)
	}
	weeklyOperating := expenseRunRate.Total / 13 * int64(10000+result.Scenario.ExpenseGrowthBPS) / 10000

	result.Adjustments = make([]Adjustment, 0)
	if err := r.db.WithContext(ctx).Raw(`SELECT a.id::text AS id,to_char(a.week_start,'YYYY-MM-DD') AS week_start,a.direction,a.label,a.amount_minor,u.email AS created_by,to_char(a.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at FROM cash_flow_adjustments a JOIN users u ON u.id=a.created_by WHERE a.organization_id=? AND a.scenario_id=? ORDER BY a.week_start,a.created_at`, organizationID, scenarioID).Scan(&result.Adjustments).Error; err != nil {
		return result, fmt.Errorf("list forecast adjustments: %w", err)
	}

	start := monday(time.Now().UTC())
	result.Weeks = make([]Week, 13)
	for index := range result.Weeks {
		result.Weeks[index].WeekStart = start.AddDate(0, 0, index*7).Format(time.DateOnly)
		result.Weeks[index].OperatingOutflowsMinor = weeklyOperating
	}
	weekIndex := func(value string) int {
		date, err := time.Parse(time.DateOnly, value)
		if err != nil {
			return -1
		}
		if date.Before(start) {
			return 0
		}
		return int(date.Sub(start).Hours() / 24 / 7)
	}
	for _, item := range items {
		index := weekIndex(item.ExpectedOn)
		if index < 0 || index >= len(result.Weeks) {
			continue
		}
		if item.Source == "invoice" {
			result.Weeks[index].InvoiceInflowsMinor += item.AmountMinor
		} else {
			result.Weeks[index].ProjectInflowsMinor += item.AmountMinor
		}
	}
	for _, item := range result.Adjustments {
		index := weekIndex(item.WeekStart)
		if index < 0 || index >= len(result.Weeks) {
			continue
		}
		if item.Direction == "inflow" {
			result.Weeks[index].ManualInflowsMinor += item.AmountMinor
		} else {
			result.Weeks[index].ManualOutflowsMinor += item.AmountMinor
		}
	}
	balance := result.OpeningCashMinor
	result.MinimumCashMinor = balance
	result.MinimumCashWeek = result.Weeks[0].WeekStart
	for index := range result.Weeks {
		week := &result.Weeks[index]
		week.OpeningBalanceMinor = balance
		inflows := week.InvoiceInflowsMinor + week.ProjectInflowsMinor + week.ManualInflowsMinor
		outflows := week.OperatingOutflowsMinor + week.ManualOutflowsMinor
		week.NetChangeMinor = inflows - outflows
		week.EndingBalanceMinor = balance + week.NetChangeMinor
		balance = week.EndingBalanceMinor
		result.TotalInflowsMinor += inflows
		result.TotalOutflowsMinor += outflows
		if balance < result.MinimumCashMinor {
			result.MinimumCashMinor, result.MinimumCashWeek = balance, week.WeekStart
		}
	}
	return result, nil
}

func (r *PostgresRepository) CreateScenario(ctx context.Context, organizationID string, actor Actor, input ScenarioInput) (Forecast, error) {
	var row struct{ ID string }
	result := r.db.WithContext(ctx).Raw(`INSERT INTO cash_flow_scenarios (organization_id,name,collection_rate_bps,collection_delay_days,expense_growth_bps,milestone_probability_bps,created_by) VALUES (?,?,?,?,?,?,?) RETURNING id::text AS id`, organizationID, input.Name, input.CollectionRateBPS, input.CollectionDelayDays, input.ExpenseGrowthBPS, input.MilestoneProbabilityBPS, actor.UserID).Scan(&row)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return Forecast{}, apperrors.ErrConflict
		}
		return Forecast{}, fmt.Errorf("create cash-flow scenario: %w", result.Error)
	}
	return r.Get(ctx, organizationID, row.ID)
}

func (r *PostgresRepository) AddAdjustment(ctx context.Context, organizationID, scenarioID string, actor Actor, input AdjustmentInput) (Forecast, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM cash_flow_scenarios WHERE id=? AND organization_id=?)`, scenarioID, organizationID).Scan(&exists).Error; err != nil {
			return fmt.Errorf("check cash-flow scenario: %w", err)
		}
		if !exists {
			return apperrors.ErrNotFound
		}
		var row struct{ ID string }
		if err := tx.Raw(`INSERT INTO cash_flow_adjustments (organization_id,scenario_id,week_start,direction,label,amount_minor,created_by) VALUES (?,?,?::date,?,?,?,?) RETURNING id::text AS id`, organizationID, scenarioID, input.WeekStart, input.Direction, input.Label, input.AmountMinor, actor.UserID).Scan(&row).Error; err != nil {
			return fmt.Errorf("insert forecast adjustment: %w", err)
		}
		metadata, err := json.Marshal(map[string]any{"adjustmentId": row.ID, "weekStart": input.WeekStart, "direction": input.Direction, "label": input.Label, "amountMinor": input.AmountMinor})
		if err != nil {
			return fmt.Errorf("encode forecast audit: %w", err)
		}
		if err := tx.Exec(`INSERT INTO audit_events (organization_id,actor,action,entity_type,entity_id,metadata) VALUES (?,?,'cash_flow.adjustment_added','cash_flow_scenario',?,?::jsonb)`, organizationID, actor.Email, scenarioID, string(metadata)).Error; err != nil {
			return fmt.Errorf("write forecast audit: %w", err)
		}
		return nil
	})
	if err != nil {
		return Forecast{}, fmt.Errorf("add forecast adjustment transaction: %w", err)
	}
	return r.Get(ctx, organizationID, scenarioID)
}
