package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// ReportRepo 报表聚合仓储（汇总口径见 docs/04 §8）。
type ReportRepo struct {
	db *gorm.DB
}

// NewReportRepo 构建报表仓储。
func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

// 不参与物理库存变动的流水类型（仅记账），汇总时排除。
const txnReportExclude = `('reservation','reservation_release')`

// InventorySummaryRow 进销存汇总行。
type InventorySummaryRow struct {
	DrugID   int64  `json:"drug_id"`
	DrugName string `json:"drug_name"`
	Opening  int64  `json:"opening"`  // 期初（LDU）
	Inbound  int64  `json:"inbound"`  // 本期入库（LDU）
	Outbound int64  `json:"outbound"` // 本期出库（LDU，正数）
	Closing  int64  `json:"closing"`  // 期末（LDU）
}

// InventorySummary 期间进销存汇总：期初 + 入 - 出 = 期末，按药品聚合（LDU 口径）。
func (r *ReportRepo) InventorySummary(ctx context.Context, start, end *time.Time) ([]InventorySummaryRow, error) {
	sql := `
		WITH txn_ldu AS (
			SELECT t.drug_id,
			       t.created_at,
			       t.quantity,
			       CASE WHEN t.is_split THEN t.quantity ELSE t.quantity * COALESCE(d.pack_size, 1) END AS ldu_qty
			FROM inventory_transactions t
			JOIN drugs d ON d.id = t.drug_id
			WHERE t.txn_type NOT IN ` + txnReportExclude + `
		)
		SELECT t.drug_id,
		       COALESCE(d.generic_name, '') AS drug_name,
		       COALESCE(SUM(CASE WHEN t.created_at < ? THEN t.ldu_qty ELSE 0 END), 0) AS opening,
		       COALESCE(SUM(CASE WHEN t.created_at >= ? AND t.created_at <= ? AND t.ldu_qty > 0 THEN t.ldu_qty ELSE 0 END), 0) AS inbound,
		       COALESCE(SUM(CASE WHEN t.created_at >= ? AND t.created_at <= ? AND t.ldu_qty < 0 THEN -t.ldu_qty ELSE 0 END), 0) AS outbound,
		       0 AS closing
		FROM txn_ldu t
		JOIN drugs d ON d.id = t.drug_id
		GROUP BY t.drug_id, d.generic_name
		ORDER BY t.drug_id`
	rows := []InventorySummaryRow{}
	if err := r.db.WithContext(ctx).Raw(sql, start, start, end, start, end).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Closing = rows[i].Opening + rows[i].Inbound - rows[i].Outbound
	}
	return rows, nil
}

// ExpiryBucketRow 效期分析行。
type ExpiryBucketRow struct {
	DrugID   int64  `json:"drug_id"`
	DrugName string `json:"drug_name"`
	BatchNo  string `json:"batch_no"`
	Expired  int64  `json:"expired"`
	In3M     int64  `json:"in_3m"`
	In6M     int64  `json:"in_6m"`
	In12M    int64  `json:"in_12m"`
	After12M int64  `json:"after_12m"`
}

// ExpiryAnalysis 按批次效期分档统计可用库存（LDU）。today 注入便于测试。
func (r *ReportRepo) ExpiryAnalysis(ctx context.Context, today time.Time) ([]ExpiryBucketRow, error) {
	rows := []ExpiryBucketRow{}
	sql := `
		SELECT i.drug_id,
		       COALESCE(d.generic_name, '') AS drug_name,
		       i.batch_no,
		       SUM(CASE WHEN i.expiry_date < ? THEN (CASE WHEN i.is_split THEN i.quantity ELSE i.quantity * COALESCE(d.pack_size,1) END) ELSE 0 END) AS expired,
		       SUM(CASE WHEN i.expiry_date >= ? AND i.expiry_date < ? THEN (CASE WHEN i.is_split THEN i.quantity ELSE i.quantity * COALESCE(d.pack_size,1) END) ELSE 0 END) AS in_3m,
		       SUM(CASE WHEN i.expiry_date >= ? AND i.expiry_date < ? THEN (CASE WHEN i.is_split THEN i.quantity ELSE i.quantity * COALESCE(d.pack_size,1) END) ELSE 0 END) AS in_6m,
		       SUM(CASE WHEN i.expiry_date >= ? AND i.expiry_date < ? THEN (CASE WHEN i.is_split THEN i.quantity ELSE i.quantity * COALESCE(d.pack_size,1) END) ELSE 0 END) AS in_12m,
		       SUM(CASE WHEN i.expiry_date >= ? THEN (CASE WHEN i.is_split THEN i.quantity ELSE i.quantity * COALESCE(d.pack_size,1) END) ELSE 0 END) AS after_12m
		FROM inventory i
		JOIN drugs d ON d.id = i.drug_id
		WHERE i.quantity > 0
		GROUP BY i.drug_id, d.generic_name, i.batch_no
		ORDER BY i.drug_id, i.batch_no`
	m3 := today.AddDate(0, 3, 0)
	m6 := today.AddDate(0, 6, 0)
	m12 := today.AddDate(0, 12, 0)
	if err := r.db.WithContext(ctx).Raw(sql, today, today, m3, m3, m6, m6, m12, m12).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SpecialUsageRow 特殊药品使用统计行。
type SpecialUsageRow struct {
	DrugID      int64  `json:"drug_id"`
	DrugName    string `json:"drug_name"`
	BatchNo     string `json:"batch_no"`
	Month       string `json:"month"`
	DispenseQty int64  `json:"dispense_qty"` // 发药数量
	ReturnQty   int64  `json:"return_qty"`   // 退回数量
}

// SpecialDrugUsage 特殊药品使用统计（按药/批号/月度聚合专账）。
func (r *ReportRepo) SpecialDrugUsage(ctx context.Context, start, end *time.Time) ([]SpecialUsageRow, error) {
	rows := []SpecialUsageRow{}
	sql := `
		SELECT l.drug_id,
		       COALESCE(d.generic_name, '') AS drug_name,
		       l.batch_no,
		       TO_CHAR(l.created_at, 'YYYY-MM') AS month,
		       SUM(CASE WHEN l.log_type = 'dispense' THEN l.quantity ELSE 0 END) AS dispense_qty,
		       SUM(CASE WHEN l.log_type = 'return' THEN -l.quantity ELSE 0 END) AS return_qty
		FROM special_drug_ledgers l
		JOIN drugs d ON d.id = l.drug_id
		WHERE l.created_at >= ? AND l.created_at <= ?
		GROUP BY l.drug_id, d.generic_name, l.batch_no, TO_CHAR(l.created_at, 'YYYY-MM')
		ORDER BY l.drug_id, month`
	if err := r.db.WithContext(ctx).Raw(sql, start, end).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// WorkloadRow 调配工作量统计行。
type WorkloadRow struct {
	DispensedBy       int64  `json:"dispensed_by"`
	DispensedName     string `json:"dispensed_name"`
	Date              string `json:"date"`
	PrescriptionCount int64  `json:"prescription_count"`
	ItemCount         int64  `json:"item_count"`
}

// DispensingWorkload 处方调配工作量（按发药人与日期聚合）。
func (r *ReportRepo) DispensingWorkload(ctx context.Context, start, end *time.Time) ([]WorkloadRow, error) {
	rows := []WorkloadRow{}
	sql := `
		SELECT rec.dispensed_by,
		       COALESCE(rec.dispensed_by_name, '') AS dispensed_name,
		       TO_CHAR(rec.created_at, 'YYYY-MM-DD') AS date,
		       COUNT(DISTINCT rec.prescription_id) AS prescription_count,
		       COUNT(*) AS item_count
		FROM prescription_dispense_records rec
		WHERE rec.created_at >= ? AND rec.created_at <= ?
		GROUP BY rec.dispensed_by, rec.dispensed_by_name, TO_CHAR(rec.created_at, 'YYYY-MM-DD')
		ORDER BY date, dispensed_name`
	if err := r.db.WithContext(ctx).Raw(sql, start, end).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SplitStatRow 拆零统计行（docs/13 F5：拆零量/损耗/毛利）。
type SplitStatRow struct {
	DrugID       int64  `json:"drug_id"`
	DrugName     string `json:"drug_name"`
	SplitBoxes   int64  `json:"split_boxes"`   // 拆零盒数（split_out）
	SplitUnits   int64  `json:"split_units"`   // 拆零入片数（split_in）
	LossUnits    int64  `json:"loss_units"`    // 拆零损耗片数 = 拆盒数×包装含量 − 入片数
	SplitCost    int64  `json:"split_cost"`    // 拆零成本（分，拆零操作单进价口径）
	SplitRevenue int64  `json:"split_revenue"` // 拆零收入（分，拆零发药记录金额）
	Margin       int64  `json:"margin"`        // 拆零毛利（分）= 收入 − 成本
}

// SplitStatistics 期间拆零统计（按药品聚合）。损耗按「应拆出片数 − 实际入片数」估算；
// 成本来自拆零操作单（split_orders），收入来自拆零发药记录（is_split=true）。
func (r *ReportRepo) SplitStatistics(ctx context.Context, start, end *time.Time) ([]SplitStatRow, error) {
	rows := []SplitStatRow{}
	sql := `
		WITH stats AS (
			SELECT t.drug_id,
			       COALESCE(SUM(CASE WHEN t.txn_type = 'split_out' THEN -t.quantity ELSE 0 END), 0) AS split_boxes,
			       COALESCE(SUM(CASE WHEN t.txn_type = 'split_in' THEN t.quantity ELSE 0 END), 0) AS split_units
			FROM inventory_transactions t
			WHERE t.txn_type IN ('split_out', 'split_in')
			  AND t.created_at >= ? AND t.created_at <= ?
			GROUP BY t.drug_id
		),
		cost AS (
			SELECT drug_id, SUM(units * split_unit_cost) AS split_cost
			FROM split_orders
			WHERE created_at >= ? AND created_at <= ?
			GROUP BY drug_id
		),
		revenue AS (
			SELECT rec.drug_id, SUM(rec.amount) AS split_revenue
			FROM prescription_dispense_records rec
			WHERE rec.is_split = TRUE AND rec.created_at >= ? AND rec.created_at <= ?
			GROUP BY rec.drug_id
		)
		SELECT s.drug_id,
		       COALESCE(d.generic_name, '') AS drug_name,
		       s.split_boxes,
		       s.split_units,
		       COALESCE(c.split_cost, 0)    AS split_cost,
		       COALESCE(r.split_revenue, 0) AS split_revenue,
		       0 AS loss_units,
		       0 AS margin
		FROM stats s
		JOIN drugs d ON d.id = s.drug_id
		LEFT JOIN cost c ON c.drug_id = s.drug_id
		LEFT JOIN revenue r ON r.drug_id = s.drug_id
		ORDER BY s.drug_id`
	if err := r.db.WithContext(ctx).Raw(sql, start, end, start, end, start, end).Scan(&rows).Error; err != nil {
		return nil, err
	}
	// 损耗 = 拆盒数 × 包装含量 − 入片数（需按药品取 pack_size）
	var packs []struct {
		DrugID   int64
		PackSize int
	}
	if err := r.db.WithContext(ctx).Model(&model.Drug{}).
		Select("id AS drug_id, pack_size").Scan(&packs).Error; err != nil {
		return nil, err
	}
	packByID := make(map[int64]int, len(packs))
	for _, p := range packs {
		packByID[p.DrugID] = p.PackSize
	}
	for i := range rows {
		ps := packByID[rows[i].DrugID]
		if ps <= 0 {
			ps = 1
		}
		loss := rows[i].SplitBoxes*int64(ps) - rows[i].SplitUnits
		if loss < 0 {
			loss = 0
		}
		rows[i].LossUnits = loss
		rows[i].Margin = rows[i].SplitRevenue - rows[i].SplitCost
	}
	return rows, nil
}
