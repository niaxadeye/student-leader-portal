package merch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/points"
)

type pointAppender interface {
	AppendTx(context.Context, pgx.Tx, points.AppendInput) (*points.Entry, bool, error)
}

type txAuditor interface {
	LogEntryTx(context.Context, pgx.Tx, audit.Entry) error
}

type Repo struct {
	pool   *pgxpool.Pool
	points pointAppender
	audit  txAuditor
}

func NewRepo(pool *pgxpool.Pool, pointRepo pointAppender, auditor txAuditor) *Repo {
	return &Repo{pool: pool, points: pointRepo, audit: auditor}
}

func (r *Repo) Can(ctx context.Context, userID, contestID, permission string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contests c WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, eventpermissions.GrantsFor(permission)).Scan(&allowed)
	return allowed, err
}

const productSelect = `
	SELECT p.id,p.contest_id,p.title,p.slug,p.description,p.price_points,
	       p.discount_price_points,p.stock_quantity,p.reserved_quantity,
	       p.stock_quantity-p.reserved_quantity,
	       COALESCE(p.discount_price_points,p.price_points),
	       (SELECT count(*)::int FROM merch_saving_targets t WHERE t.product_id=p.id),
	       ($2::uuid IS NOT NULL AND EXISTS(
	         SELECT 1 FROM merch_saving_targets t
	         WHERE t.product_id=p.id AND t.event_participant_id=$2::uuid)),
	       p.status,p.created_at,p.updated_at
	FROM merch_products p `

func (r *Repo) ListProducts(ctx context.Context, contestID string, participantID *string, admin bool) ([]Product, error) {
	rows, err := r.pool.Query(ctx, productSelect+`
		WHERE p.contest_id=$1 AND ($3 OR p.status IN ('ACTIVE','SOLD_OUT'))
		ORDER BY p.created_at DESC`, contestID, participantID, admin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		if err := r.loadImages(ctx, product); err != nil {
			return nil, err
		}
		products = append(products, *product)
	}
	return products, rows.Err()
}

func (r *Repo) ProductByID(ctx context.Context, contestID, productID string) (*Product, error) {
	product, err := scanProduct(r.pool.QueryRow(ctx, productSelect+`
		WHERE p.contest_id=$1 AND p.id=$3`, contestID, nil, productID))
	if err != nil {
		return nil, err
	}
	return product, r.loadImages(ctx, product)
}

func (r *Repo) ProductBySlug(ctx context.Context, contestID, slug, participantID string) (*Product, error) {
	product, err := scanProduct(r.pool.QueryRow(ctx, productSelect+`
		WHERE p.contest_id=$1 AND p.slug=$3 AND p.status IN ('ACTIVE','SOLD_OUT')`,
		contestID, participantID, slug))
	if err != nil {
		return nil, err
	}
	return product, r.loadImages(ctx, product)
}

func (r *Repo) CreateProduct(ctx context.Context, contestID, slugBase string, input ProductInput) (*Product, error) {
	for suffix := 1; suffix <= 100; suffix++ {
		slug := slugBase
		if suffix > 1 {
			slug = fmt.Sprintf("%s-%d", slugBase, suffix)
		}
		var id string
		err := r.pool.QueryRow(ctx, `
			INSERT INTO merch_products
			  (contest_id,title,slug,description,price_points,discount_price_points,stock_quantity)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, contestID, input.Title, slug,
			input.Description, input.PricePoints, input.DiscountPricePoints, input.StockQuantity).Scan(&id)
		if err == nil {
			return r.ProductByID(ctx, contestID, id)
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != "uq_merch_products_slug" {
			return nil, err
		}
	}
	return nil, ErrValidation
}

func (r *Repo) UpdateProduct(ctx context.Context, contestID, productID string, input ProductInput) (*Product, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE merch_products SET title=$3,description=$4,price_points=$5,
		  discount_price_points=$6,stock_quantity=$7,
		  status=CASE WHEN status IN ('ACTIVE','SOLD_OUT') THEN
		    CASE WHEN $7-reserved_quantity=0 THEN 'SOLD_OUT' ELSE 'ACTIVE' END
		    ELSE status END,updated_at=now()
		WHERE contest_id=$1 AND id=$2 AND $7>=reserved_quantity`,
		contestID, productID, input.Title, input.Description, input.PricePoints,
		input.DiscountPricePoints, input.StockQuantity)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM merch_products
			WHERE contest_id=$1 AND id=$2)`, contestID, productID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
		return nil, ErrInsufficientStock
	}
	return r.ProductByID(ctx, contestID, productID)
}

func (r *Repo) TransitionProduct(ctx context.Context, contestID, productID, action string) (*Product, error) {
	var command pgconn.CommandTag
	var err error
	if action == "activate" {
		command, err = r.pool.Exec(ctx, `UPDATE merch_products SET
			status=CASE WHEN stock_quantity=reserved_quantity THEN 'SOLD_OUT' ELSE 'ACTIVE' END,
			updated_at=now() WHERE contest_id=$1 AND id=$2
			AND status IN ('DRAFT','HIDDEN','SOLD_OUT')`, contestID, productID)
	} else {
		command, err = r.pool.Exec(ctx, `UPDATE merch_products SET status='HIDDEN',updated_at=now()
			WHERE contest_id=$1 AND id=$2 AND status IN ('ACTIVE','SOLD_OUT')`, contestID, productID)
	}
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, r.notFoundOrTransition(ctx, contestID, productID)
	}
	return r.ProductByID(ctx, contestID, productID)
}

func (r *Repo) DeleteProduct(ctx context.Context, contestID, productID string) (keys []string, err error) {
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM merch_products
			WHERE contest_id=$1 AND id=$2 FOR UPDATE`, contestID, productID).Scan(&status); err != nil {
			return mapNotFound(err)
		}
		if status != ProductDraft {
			return ErrInvalidTransition
		}
		var used bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM merch_order_items WHERE product_id=$1
			UNION ALL SELECT 1 FROM merch_saving_targets WHERE product_id=$1)`, productID).Scan(&used); err != nil {
			return err
		}
		if used {
			return ErrInvalidTransition
		}
		rows, err := tx.Query(ctx, `DELETE FROM merch_product_images
			WHERE contest_id=$1 AND product_id=$2 RETURNING object_key`, contestID, productID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				return err
			}
			keys = append(keys, key)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		_, err = tx.Exec(ctx, `DELETE FROM merch_products WHERE contest_id=$1 AND id=$2`, contestID, productID)
		return err
	})
	return keys, err
}

func (r *Repo) AddImage(ctx context.Context, contestID, productID string, image ProductImage) (*ProductImage, error) {
	created, err := scanImage(r.pool.QueryRow(ctx, `
		INSERT INTO merch_product_images
		  (contest_id,product_id,object_key,original_name,mime_type,size_bytes,sort_order)
		SELECT p.contest_id,p.id,$3,$4,$5,$6,$7 FROM merch_products p
		WHERE p.contest_id=$1 AND p.id=$2
		RETURNING id,product_id,object_key,original_name,mime_type,size_bytes,sort_order,created_at`,
		contestID, productID, image.ObjectKey, image.OriginalName, image.MimeType,
		image.SizeBytes, image.SortOrder))
	return created, err
}

func (r *Repo) DeleteImage(ctx context.Context, contestID, productID, imageID string) (*ProductImage, error) {
	return scanImage(r.pool.QueryRow(ctx, `DELETE FROM merch_product_images
		WHERE contest_id=$1 AND product_id=$2 AND id=$3
		RETURNING id,product_id,object_key,original_name,mime_type,size_bytes,sort_order,created_at`,
		contestID, productID, imageID))
}

func (r *Repo) SetSavingTarget(ctx context.Context, contestID, participantID, productID string) (*Product, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var participantStatus, contestStatus, productStatus string
		if err := tx.QueryRow(ctx, `
			SELECT ep.status,c.status,mp.status
			FROM event_participants ep
			JOIN contests c ON c.id=ep.contest_id
			JOIN merch_products mp ON mp.contest_id=ep.contest_id AND mp.id=$3
			WHERE ep.contest_id=$1 AND ep.id=$2 FOR UPDATE OF ep`,
			contestID, participantID, productID).
			Scan(&participantStatus, &contestStatus, &productStatus); err != nil {
			return mapNotFound(err)
		}
		if participantStatus != "ACTIVE" || contestStatus != "ACTIVE" ||
			(productStatus != ProductActive && productStatus != ProductSoldOut) {
			return ErrInvalidTransition
		}
		if _, err := tx.Exec(ctx, `INSERT INTO merch_saving_targets
			(contest_id,event_participant_id,product_id) VALUES ($1,$2,$3)
			ON CONFLICT (event_participant_id) DO UPDATE SET
			contest_id=EXCLUDED.contest_id,product_id=EXCLUDED.product_id,created_at=now()`,
			contestID, participantID, productID); err != nil {
			return err
		}
		return r.auditParticipant(ctx, tx, participantID, contestID, "MERCH_SAVING_TARGET_SET",
			"merch_product", productID, map[string]any{"product_id": productID})
	})
	if err != nil {
		return nil, err
	}
	return r.ProductBySlugForParticipantID(ctx, contestID, productID, participantID)
}

func (r *Repo) ProductBySlugForParticipantID(ctx context.Context, contestID, productID, participantID string) (*Product, error) {
	product, err := scanProduct(r.pool.QueryRow(ctx, productSelect+`
		WHERE p.contest_id=$1 AND p.id=$3`, contestID, participantID, productID))
	if err != nil {
		return nil, err
	}
	return product, r.loadImages(ctx, product)
}

func (r *Repo) DeleteSavingTarget(ctx context.Context, contestID, participantID string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var targetID, productID string
		err := tx.QueryRow(ctx, `DELETE FROM merch_saving_targets
			WHERE contest_id=$1 AND event_participant_id=$2 RETURNING id,product_id`,
			contestID, participantID).Scan(&targetID, &productID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return r.auditParticipant(ctx, tx, participantID, contestID, "MERCH_SAVING_TARGET_DELETED",
			"merch_product", productID, map[string]any{"target_id": targetID})
	})
}

type reservedProduct struct {
	ID, Title, Status               string
	Price, Discount                 *int64
	StockQuantity, ReservedQuantity int
}

func (r *Repo) Reserve(ctx context.Context, params ReserveParams) (result *OrderResult, err error) {
	replayed := false
	var orderID string
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// The advisory lock makes the contest-wide idempotency key deterministic even
		// when two different participants submit the same key concurrently.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
			params.ContestID+":"+params.IdempotencyKey); err != nil {
			return err
		}
		var existingParticipant, existingFingerprint string
		existingErr := tx.QueryRow(ctx, `SELECT id,event_participant_id,request_fingerprint
			FROM merch_orders WHERE contest_id=$1 AND idempotency_key=$2`,
			params.ContestID, params.IdempotencyKey).
			Scan(&orderID, &existingParticipant, &existingFingerprint)
		if existingErr == nil {
			if existingParticipant != params.EventParticipantID || existingFingerprint != params.RequestFingerprint {
				return ErrIdempotencyConflict
			}
			replayed = true
			return nil
		}
		if !errors.Is(existingErr, pgx.ErrNoRows) {
			return existingErr
		}

		var participantStatus, contestStatus string
		if err := tx.QueryRow(ctx, `SELECT ep.status,c.status
			FROM event_participants ep JOIN contests c ON c.id=ep.contest_id
			WHERE ep.contest_id=$1 AND ep.id=$2 FOR UPDATE OF ep`,
			params.ContestID, params.EventParticipantID).
			Scan(&participantStatus, &contestStatus); err != nil {
			return mapNotFound(err)
		}
		if participantStatus != "ACTIVE" || contestStatus != "ACTIVE" {
			return ErrInvalidTransition
		}

		ids := make([]string, len(params.Items))
		quantities := make(map[string]int, len(params.Items))
		for i, item := range params.Items {
			ids[i], quantities[item.ProductID] = item.ProductID, item.Quantity
		}
		rows, err := tx.Query(ctx, `SELECT id,title,status,price_points,discount_price_points,
			stock_quantity,reserved_quantity FROM merch_products
			WHERE contest_id=$1 AND id=ANY($2::uuid[]) ORDER BY id FOR UPDATE`, params.ContestID, ids)
		if err != nil {
			return err
		}
		products := make([]reservedProduct, 0, len(ids))
		for rows.Next() {
			var product reservedProduct
			var price int64
			if err := rows.Scan(&product.ID, &product.Title, &product.Status, &price,
				&product.Discount, &product.StockQuantity, &product.ReservedQuantity); err != nil {
				rows.Close()
				return err
			}
			product.Price = &price
			products = append(products, product)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(products) != len(ids) {
			return ErrNotFound
		}

		var total int64
		prices := make(map[string]int64, len(products))
		for _, product := range products {
			quantity := quantities[product.ID]
			if product.Status != ProductActive || product.StockQuantity-product.ReservedQuantity < quantity {
				return ErrInsufficientStock
			}
			price := *product.Price
			if product.Discount != nil {
				price = *product.Discount
			}
			prices[product.ID] = price
			total += price * int64(quantity)
		}
		var ledger, held int64
		if err := tx.QueryRow(ctx, `SELECT
			COALESCE((SELECT sum(amount) FROM points_ledger
			 WHERE contest_id=$1 AND event_participant_id=$2),0)::bigint,
			COALESCE((SELECT sum(amount) FROM points_holds
			 WHERE contest_id=$1 AND event_participant_id=$2 AND status='ACTIVE'),0)::bigint`,
			params.ContestID, params.EventParticipantID).Scan(&ledger, &held); err != nil {
			return err
		}
		if total > ledger-held {
			return ErrInsufficientPoints
		}
		if err := tx.QueryRow(ctx, `INSERT INTO merch_orders
			(contest_id,event_participant_id,status,points_total,idempotency_key,request_fingerprint,created_at,updated_at)
			VALUES ($1,$2,'RESERVED',$3,$4,$5,$6,$6) RETURNING id`, params.ContestID,
			params.EventParticipantID, total, params.IdempotencyKey, params.RequestFingerprint,
			params.Now).Scan(&orderID); err != nil {
			return err
		}
		for _, product := range products {
			quantity, price := quantities[product.ID], prices[product.ID]
			if _, err := tx.Exec(ctx, `INSERT INTO merch_order_items
				(contest_id,order_id,product_id,product_title,quantity,price_points,total_points)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`, params.ContestID, orderID, product.ID,
				product.Title, quantity, price, price*int64(quantity)); err != nil {
				return err
			}
			command, err := tx.Exec(ctx, `UPDATE merch_products SET
				reserved_quantity=reserved_quantity+$3,
				status=CASE WHEN stock_quantity-reserved_quantity-$3=0 THEN 'SOLD_OUT' ELSE status END,
				updated_at=now() WHERE contest_id=$1 AND id=$2 AND status='ACTIVE'
				AND stock_quantity-reserved_quantity >= $3`, params.ContestID, product.ID, quantity)
			if err != nil {
				return err
			}
			if command.RowsAffected() != 1 {
				return ErrInsufficientStock
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO points_holds
			(contest_id,event_participant_id,merch_order_id,amount)
			VALUES ($1,$2,$3,$4)`, params.ContestID, params.EventParticipantID, orderID, total); err != nil {
			return err
		}
		return r.auditParticipant(ctx, tx, params.EventParticipantID, params.ContestID,
			"MERCH_ORDER_RESERVED", "merch_order", orderID,
			map[string]any{"points": total, "items_count": len(products)})
	})
	if err != nil {
		return nil, err
	}
	order, err := r.ParticipantOrder(ctx, params.ContestID, params.EventParticipantID, orderID)
	if err != nil {
		return nil, err
	}
	return &OrderResult{Order: *order, Replayed: replayed}, nil
}

func (r *Repo) ParticipantOrders(ctx context.Context, contestID, participantID string) ([]Order, error) {
	return r.listOrders(ctx, orderSelect+` WHERE o.contest_id=$1 AND o.event_participant_id=$2
		ORDER BY o.created_at DESC`, contestID, participantID)
}

func (r *Repo) ParticipantOrder(ctx context.Context, contestID, participantID, orderID string) (*Order, error) {
	order, err := scanOrder(r.pool.QueryRow(ctx, orderSelect+`
		WHERE o.contest_id=$1 AND o.event_participant_id=$2 AND o.id=$3`,
		contestID, participantID, orderID))
	if err != nil {
		return nil, err
	}
	return order, r.loadOrderItems(ctx, order)
}

func (r *Repo) AdminOrders(ctx context.Context, contestID, status string) ([]Order, error) {
	return r.listOrders(ctx, orderSelect+` WHERE o.contest_id=$1 AND ($2='' OR o.status=$2)
		ORDER BY o.created_at DESC`, contestID, status)
}

func (r *Repo) AdminOrder(ctx context.Context, contestID, orderID string) (*Order, error) {
	order, err := scanOrder(r.pool.QueryRow(ctx, orderSelect+`
		WHERE o.contest_id=$1 AND o.id=$2`, contestID, orderID))
	if err != nil {
		return nil, err
	}
	return order, r.loadOrderItems(ctx, order)
}

func (r *Repo) Issue(ctx context.Context, actor Actor, contestID, orderID string) (result *OrderResult, err error) {
	replayed := false
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var participantID, status string
		var total int64
		if err := tx.QueryRow(ctx, `SELECT event_participant_id,status,points_total FROM merch_orders
			WHERE contest_id=$1 AND id=$2 FOR UPDATE`, contestID, orderID).
			Scan(&participantID, &status, &total); err != nil {
			return mapNotFound(err)
		}
		if status == OrderIssued {
			replayed = true
			return nil
		}
		if status != OrderReserved {
			return ErrInvalidTransition
		}
		items, err := orderItemsTx(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := lockOrderProducts(ctx, tx, contestID, items); err != nil {
			return err
		}
		for _, item := range items {
			command, err := tx.Exec(ctx, `UPDATE merch_products SET
				stock_quantity=stock_quantity-$3,reserved_quantity=reserved_quantity-$3,
				status=CASE WHEN status='HIDDEN' THEN 'HIDDEN'
				  WHEN stock_quantity-reserved_quantity=0 THEN 'SOLD_OUT' ELSE 'ACTIVE' END,
				updated_at=now() WHERE contest_id=$1 AND id=$2
				AND stock_quantity >= $3 AND reserved_quantity >= $3`,
				contestID, item.ProductID, item.Quantity)
			if err != nil {
				return err
			}
			if command.RowsAffected() != 1 {
				return ErrInsufficientStock
			}
		}
		command, err := tx.Exec(ctx, `UPDATE points_holds SET status='CAPTURED',captured_at=now()
			WHERE merch_order_id=$1 AND status='ACTIVE'`, orderID)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidTransition
		}
		sourceType, sourceID, actorID := "merch_order", orderID, actor.UserID
		_, created, err := r.points.AppendTx(ctx, tx, points.AppendInput{
			ContestID: contestID, EventParticipantID: participantID, Amount: -total,
			Type: points.TypeMerchPurchase, SourceType: &sourceType, SourceID: &sourceID,
			Description: "Покупка мерча", CreatedByUserID: &actorID,
			IdempotencyKey: "merch-purchase:" + orderID,
		})
		if err != nil {
			return err
		}
		if !created {
			return points.ErrIdempotencyConflict
		}
		command, err = tx.Exec(ctx, `UPDATE merch_orders SET status='ISSUED',issued_at=now(),
			issued_by_user_id=$3,updated_at=now() WHERE contest_id=$1 AND id=$2 AND status='RESERVED'`,
			contestID, orderID, actor.UserID)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidTransition
		}
		return r.auditStaff(ctx, tx, actor, contestID, "MERCH_ORDER_ISSUED", orderID,
			map[string]any{"participant_id": participantID, "points": total})
	})
	if err != nil {
		return nil, err
	}
	order, err := r.AdminOrder(ctx, contestID, orderID)
	if err != nil {
		return nil, err
	}
	return &OrderResult{Order: *order, Replayed: replayed}, nil
}

func (r *Repo) Reject(ctx context.Context, actor Actor, contestID, orderID, reason string) (*OrderResult, error) {
	return r.release(ctx, contestID, "", orderID, OrderRejected, reason, actor)
}

func (r *Repo) Cancel(ctx context.Context, contestID, participantID, orderID string) (*OrderResult, error) {
	return r.release(ctx, contestID, participantID, orderID, OrderCancelled, "", Actor{})
}

func (r *Repo) release(
	ctx context.Context,
	contestID, participantScope, orderID, target, reason string,
	actor Actor,
) (result *OrderResult, err error) {
	replayed := false
	var participantID string
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var status string
		query := `SELECT event_participant_id,status FROM merch_orders WHERE contest_id=$1 AND id=$2`
		args := []any{contestID, orderID}
		if participantScope != "" {
			query += ` AND event_participant_id=$3`
			args = append(args, participantScope)
		}
		query += ` FOR UPDATE`
		if err := tx.QueryRow(ctx, query, args...).Scan(&participantID, &status); err != nil {
			return mapNotFound(err)
		}
		if status == target {
			replayed = true
			return nil
		}
		if status != OrderReserved {
			return ErrInvalidTransition
		}
		items, err := orderItemsTx(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := lockOrderProducts(ctx, tx, contestID, items); err != nil {
			return err
		}
		for _, item := range items {
			command, err := tx.Exec(ctx, `UPDATE merch_products SET
				reserved_quantity=reserved_quantity-$3,
				status=CASE WHEN status='SOLD_OUT' AND stock_quantity-(reserved_quantity-$3)>0
				  THEN 'ACTIVE' ELSE status END,updated_at=now()
				WHERE contest_id=$1 AND id=$2 AND reserved_quantity >= $3`,
				contestID, item.ProductID, item.Quantity)
			if err != nil {
				return err
			}
			if command.RowsAffected() != 1 {
				return ErrInvalidTransition
			}
		}
		command, err := tx.Exec(ctx, `UPDATE points_holds SET status='RELEASED',released_at=now()
			WHERE merch_order_id=$1 AND status='ACTIVE'`, orderID)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidTransition
		}
		if target == OrderRejected {
			command, err = tx.Exec(ctx, `UPDATE merch_orders SET status='REJECTED',rejection_reason=$3,
				rejected_at=now(),rejected_by_user_id=$4,updated_at=now()
				WHERE contest_id=$1 AND id=$2 AND status='RESERVED'`,
				contestID, orderID, reason, actor.UserID)
		} else {
			command, err = tx.Exec(ctx, `UPDATE merch_orders SET status='CANCELLED',cancelled_at=now(),
				updated_at=now() WHERE contest_id=$1 AND id=$2 AND status='RESERVED'`, contestID, orderID)
		}
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrInvalidTransition
		}
		if target == OrderRejected {
			return r.auditStaff(ctx, tx, actor, contestID, "MERCH_ORDER_REJECTED", orderID,
				map[string]any{"participant_id": participantID, "reason": reason})
		}
		return r.auditParticipant(ctx, tx, participantID, contestID, "MERCH_ORDER_CANCELLED",
			"merch_order", orderID, nil)
	})
	if err != nil {
		return nil, err
	}
	var order *Order
	if participantScope != "" {
		order, err = r.ParticipantOrder(ctx, contestID, participantScope, orderID)
	} else {
		order, err = r.AdminOrder(ctx, contestID, orderID)
	}
	if err != nil {
		return nil, err
	}
	return &OrderResult{Order: *order, Replayed: replayed}, nil
}

const orderSelect = `
	SELECT o.id,o.contest_id,o.event_participant_id,p.full_name,o.status,o.points_total,
	       o.rejection_reason,o.created_at,o.updated_at,o.issued_at,o.rejected_at,o.cancelled_at,
	       o.issued_by_user_id,o.rejected_by_user_id
	FROM merch_orders o JOIN event_participants p ON p.id=o.event_participant_id `

func (r *Repo) listOrders(ctx context.Context, query string, args ...any) ([]Order, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	orders := make([]Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		orders = append(orders, *order)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range orders {
		if err := r.loadOrderItems(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (r *Repo) loadOrderItems(ctx context.Context, order *Order) error {
	rows, err := r.pool.Query(ctx, `SELECT id,product_id,product_title,quantity,price_points,total_points
		FROM merch_order_items WHERE order_id=$1 ORDER BY id`, order.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	order.Items = make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductTitle, &item.Quantity,
			&item.PricePoints, &item.TotalPoints); err != nil {
			return err
		}
		order.Items = append(order.Items, item)
	}
	return rows.Err()
}

func orderItemsTx(ctx context.Context, tx pgx.Tx, orderID string) ([]OrderItem, error) {
	rows, err := tx.Query(ctx, `SELECT id,product_id,product_title,quantity,price_points,total_points
		FROM merch_order_items WHERE order_id=$1 ORDER BY product_id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductTitle, &item.Quantity,
			&item.PricePoints, &item.TotalPoints); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

func lockOrderProducts(ctx context.Context, tx pgx.Tx, contestID string, items []OrderItem) error {
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ProductID
	}
	rows, err := tx.Query(ctx, `SELECT id FROM merch_products WHERE contest_id=$1
		AND id=ANY($2::uuid[]) ORDER BY id FOR UPDATE`, contestID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) loadImages(ctx context.Context, product *Product) error {
	rows, err := r.pool.Query(ctx, `SELECT id,product_id,object_key,original_name,mime_type,
		size_bytes,sort_order,created_at FROM merch_product_images
		WHERE product_id=$1 ORDER BY sort_order,created_at`, product.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	product.Images = make([]ProductImage, 0)
	for rows.Next() {
		image, err := scanImage(rows)
		if err != nil {
			return err
		}
		product.Images = append(product.Images, *image)
	}
	return rows.Err()
}

func (r *Repo) notFoundOrTransition(ctx context.Context, contestID, productID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM merch_products
		WHERE contest_id=$1 AND id=$2)`, contestID, productID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrInvalidTransition
}

func (r *Repo) auditParticipant(
	ctx context.Context, tx pgx.Tx, participantID, contestID, action, entityType, entityID string,
	metadata map[string]any,
) error {
	if r.audit == nil {
		return nil
	}
	return r.audit.LogEntryTx(ctx, tx, audit.Entry{
		EventParticipantID: participantID, ContestID: contestID, Action: action,
		EntityType: entityType, EntityID: entityID, Metadata: metadata,
	})
}

func (r *Repo) auditStaff(
	ctx context.Context, tx pgx.Tx, actor Actor, contestID, action, orderID string,
	metadata map[string]any,
) error {
	if r.audit == nil {
		return nil
	}
	return r.audit.LogEntryTx(ctx, tx, audit.Entry{
		ActorUserID: actor.UserID, ContestID: contestID, Action: action,
		EntityType: "merch_order", EntityID: orderID, Metadata: metadata,
	})
}

type rowScanner interface {
	Scan(...any) error
}

func scanProduct(row rowScanner) (*Product, error) {
	var product Product
	err := row.Scan(&product.ID, &product.ContestID, &product.Title, &product.Slug,
		&product.Description, &product.PricePoints, &product.DiscountPricePoints,
		&product.StockQuantity, &product.ReservedQuantity, &product.AvailableQuantity,
		&product.EffectivePrice, &product.InterestedCount, &product.IsSavingTarget,
		&product.Status, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return nil, mapNotFound(err)
	}
	product.Images = []ProductImage{}
	return &product, nil
}

func scanImage(row rowScanner) (*ProductImage, error) {
	var image ProductImage
	if err := row.Scan(&image.ID, &image.ProductID, &image.ObjectKey, &image.OriginalName,
		&image.MimeType, &image.SizeBytes, &image.SortOrder, &image.CreatedAt); err != nil {
		return nil, mapNotFound(err)
	}
	return &image, nil
}

func scanOrder(row rowScanner) (*Order, error) {
	var order Order
	err := row.Scan(&order.ID, &order.ContestID, &order.EventParticipantID,
		&order.ParticipantName, &order.Status, &order.PointsTotal, &order.RejectionReason,
		&order.CreatedAt, &order.UpdatedAt, &order.IssuedAt, &order.RejectedAt,
		&order.CancelledAt, &order.IssuedByUserID, &order.RejectedByUserID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	order.Items = []OrderItem{}
	return &order, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
