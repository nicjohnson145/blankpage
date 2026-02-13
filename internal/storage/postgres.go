package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/lib/pq"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/hlp"
	hsqlx "github.com/nicjohnson145/hlp/sqlx"
)

// Taken from https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	PGError_ForeignKeyViolation      = "23503"
	PGError_CheckConstraintViolation = "23514"
)

type PostgresConfig struct {
	DB *sql.DB
}

func NewPostgres(conf PostgresConfig) *Postgres {
	return &Postgres{
		db: sqlx.NewDb(conf.DB, "postgres"),
	}
}

type Postgres struct {
	db *sqlx.DB
}

func (p *Postgres) Purge(ctx context.Context) error {
	tables := []string{
		"blankpage_book_metadata",
		"blankpage_book_content",
		"blankpage_shelf",
		"blankpage_shelf_member",
	}
	return hsqlx.WithTransaction(p.db, func(txn *sqlx.Tx) error {
		for _, table := range tables {
			if _, err := txn.ExecContext(ctx, fmt.Sprintf("DELETE FROM %v", table)); err != nil {
				return fmt.Errorf("error deleting: %w", err)
			}
		}
		return nil
	})
}

func (p *Postgres) AddBook(ctx context.Context, req *pbv1.AddBookRequest) error {
	return hsqlx.WithTransaction(p.db, func(txn *sqlx.Tx) error {
		stmt := `
			INSERT INTO
				blankpage_book_metadata
				(
					id,
					title,
					author,
					series,
					series_number,
					upload_ts
				)
			VALUES
				(
					:id,
					:title,
					:author,
					:series,
					:series_number,
					:upload_ts
				)
			ON CONFLICT ON CONSTRAINT
				blankpage_book_metadata_pk
			DO
				UPDATE
			SET
				title = EXCLUDED.title,
				author = EXCLUDED.author,
				series = EXCLUDED.series,
				series_number = EXCLUDED.series_number
		`
		if _, err := txn.NamedExecContext(ctx, stmt, PBMetadataToDBBookMetadata(req.Book.Metadata)); err != nil {
			return fmt.Errorf("error inserting metadata: %w", err)
		}

		stmt = `
			INSERT INTO
				blankpage_book_content
				(
					book_id,
					format,
					content
				)
			VALUES
				(
					:book_id,
					:format,
					:content
				)
			ON CONFLICT ON CONSTRAINT
				blankpage_book_content_pk
			DO
				UPDATE
			SET
				format = EXCLUDED.format,
				content = EXCLUDED.content
		`
		arg := DBBookContent{
			BookID: req.Book.Metadata.Id,
			// TODO: this should be inferred or specified somehow
			Format:  "epub",
			Content: req.Book.Content,
		}
		if _, err := txn.NamedExecContext(ctx, stmt, arg); err != nil {
			return fmt.Errorf("error inserting content: %w", err)
		}

		return nil
	})
}

func (p *Postgres) ReadBook(ctx context.Context, id string) (*pbv1.Book, error) {
	stmt := `
		SELECT
			bm.*,
			bc.content
		FROM
			blankpage_book_metadata AS bm
		JOIN
			blankpage_book_content AS bc
		ON
			bm.id = bc.book_id
		WHERE
			bm.id = :book_id AND
			bc.format = :format
	`
	args := map[string]any{
		"book_id": id,
		// TODO: probably request based with a default
		"format": "epub",
	}

	rows, err := hsqlx.RequireExactSelectNamedCtx[DBOutputBook](ctx, 1, p.db, stmt, args)
	if err != nil {
		if errors.Is(err, hsqlx.ErrNotFoundError) {
			return nil, ErrUnknownBookError
		}
		return nil, fmt.Errorf("error selecting: %w", err)
	}

	return DBOutputBookToPBBook(rows[0]), nil
}

func (p *Postgres) RemoveBook(ctx context.Context, req *pbv1.RemoveBookRequest) error {
	stmt := `
		DELETE FROM
			blankpage_book_metadata
		WHERE
			id = :book_id
	`
	args := map[string]any{
		"book_id": req.BookId,
	}

	if _, err := p.db.NamedExecContext(ctx, stmt, args); err != nil {
		return fmt.Errorf("error deleting: %w", err)
	}
	return nil
}

func (p *Postgres) ListMetadata(ctx context.Context, req *pbv1.ListBooksRequest) (*ListMetadataResponse, error) {
	return hsqlx.WithTransactionReturning(p.db, func(txn *sqlx.Tx) (*ListMetadataResponse, error) {
		psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

		query := psql.Select("*").From("blankpage_book_metadata")

		var orderField string
		switch req.SortOptions.SortField {
		case pbv1.ListBooksRequest_SortOptions_SORT_FIELD_TITLE:
			orderField = "title"
		case pbv1.ListBooksRequest_SortOptions_SORT_FIELD_UPLOAD_DATE:
			orderField = "upload_ts"
		default:
			return nil, fmt.Errorf("unhandled sort field %v", req.SortOptions.SortField)
		}

		var orderDirection string
		switch req.SortOptions.SortOrder {
		case pbv1.ListBooksRequest_SortOptions_SORT_ORDER_ASCENDING:
			orderDirection = "ASC"
		case pbv1.ListBooksRequest_SortOptions_SORT_ORDER_DESCENDING:
			orderDirection = "DESC"
		default:
			return nil, fmt.Errorf("unhandled sort order %v", req.SortOptions.SortOrder)
		}

		query = query.
			OrderBy(orderField + " " + orderDirection).
			Offset(uint64(int(*req.PaginationOptions.Page) * int(*req.PaginationOptions.PerPage))).
			Limit(uint64(*req.PaginationOptions.PerPage))

		stmt, _, err := query.ToSql()
		if err != nil {
			return nil, fmt.Errorf("error building sql: %w", err)
		}

		rows, err := hsqlx.SelectCtx[DBBookMetadata](ctx, txn, stmt)
		if err != nil {
			return nil, fmt.Errorf("error selecting: %w", err)
		}

		outRows := hlp.Map(rows, func(r DBBookMetadata, _ int) *pbv1.Metadata {
			return DBBookMetadataToPBMetadata(r)
		})

		stmt = `
			SELECT
				COUNT(*)
			FROM
				blankpage_book_metadata
		`
		type countRow struct {
			Count int `db:"count"`
		}

		countRows, err := hsqlx.SelectCtx[countRow](ctx, txn, stmt)
		if err != nil {
			return nil, fmt.Errorf("error selecting row count: %w", err)
		}

		lower := int(*req.PaginationOptions.Page * *req.PaginationOptions.PerPage)
		upper := int(math.Min(float64(lower+int(*req.PaginationOptions.PerPage)), float64(countRows[0].Count)))

		return &ListMetadataResponse{
			Books:   outRows,
			HasMore: upper < countRows[0].Count,
		}, nil
	})
}

func (p *Postgres) GetAllSeries(ctx context.Context) ([]string, error) {
	stmt := `
		SELECT
			DISTINCT(series)
		FROM
			blankpage_book_metadata
		WHERE
			series IS NOT NULL
	`
	type row struct {
		Series string `db:"series"`
	}

	rows, err := hsqlx.SelectCtx[row](ctx, p.db, stmt)
	if err != nil {
		return nil, fmt.Errorf("error selecting: %w", err)
	}

	return hlp.Map(rows, func(r row, _ int) string {
		return r.Series
	}), nil
}

func (p *Postgres) GetBooksBySeries(ctx context.Context, series string) ([]*pbv1.Metadata, error) {
	stmt := `
		SELECT
			*
		FROM
			blankpage_book_metadata
		WHERE
			series = :series
	`
	args := map[string]any{
		"series": series,
	}

	rows, err := hsqlx.SelectNamedCtx[DBBookMetadata](ctx, p.db, stmt, args)
	if err != nil {
		return nil, fmt.Errorf("error selecting: %w", err)
	}

	return hlp.Map(rows, func(r DBBookMetadata, _ int) *pbv1.Metadata {
		return DBBookMetadataToPBMetadata(r)
	}), nil
}

func (p *Postgres) CreateShelf(ctx context.Context, shelf *pbv1.Shelf) error {
	stmt := `
		INSERT INTO
			blankpage_shelf
			(
				id,
				name
			)
		VALUES
			(
				:id,
				:name
			)
	`
	if _, err := p.db.NamedExecContext(ctx, stmt, PBShelfToDBShelf(shelf)); err != nil {
		return fmt.Errorf("error inserting: %w", err)
	}

	return nil
}

func (p *Postgres) ListShelves(ctx context.Context) ([]*pbv1.Shelf, error) {
	stmt := `
		SELECT
			*
		FROM
			blankpage_shelf
		ORDER BY
			id DESC
	`
	rows, err := hsqlx.SelectCtx[DBShelf](ctx, p.db, stmt)
	if err != nil {
		return nil, fmt.Errorf("error selecting: %w", err)
	}

	return hlp.Map(rows, func(r DBShelf, _ int) *pbv1.Shelf {
		return DBShelfToPBShelf(r)
	}), nil
}

func (p *Postgres) AddBooksToShelf(ctx context.Context, req *pbv1.AddBooksToShelfRequest) error {
	stmt := `
		INSERT INTO
			blankpage_shelf_member
			(
				shelf_id,
				book_id
			)
		VALUES
			(
				:shelf_id,
				:book_id
			)
	`
	if _, err := p.db.NamedExecContext(ctx, stmt, PBAddShelfBooksRequestToDBShelfMembers(req)); err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case PGError_ForeignKeyViolation:
				switch pgErr.Constraint {
				case "shelf_id_fk":
					return ErrUnknownShelfError
				case "book_id_fk":
					return ErrUnknownBookError
				}
			}
		}
		return fmt.Errorf("error inserting: %w", err)
	}

	return nil
}

func (p *Postgres) RemoveBooksFromShelf(ctx context.Context, req *pbv1.RemoveBooksFromShelfRequest) error {
	stmt := `
		DELETE FROM
			blankpage_shelf_member
		WHERE
			shelf_id = :shelf_id AND
			book_id = ANY(:book_ids)
	`
	args := map[string]any{
		"shelf_id": req.ShelfId,
		"book_ids": pq.Array(req.BookIds),
	}

	if _, err := p.db.NamedExecContext(ctx, stmt, args); err != nil {
		return fmt.Errorf("error deleting: %w", err)
	}

	return nil
}

func (p *Postgres) ListBooksForShelf(ctx context.Context, req *pbv1.ListBooksForShelfRequest) ([]*pbv1.Metadata, error) {
	stmt := `
		SELECT
			bm.*
		FROM
			blankpage_shelf_member AS sm
		JOIN
			blankpage_book_metadata AS bm
		ON
			sm.book_id = bm.id
		WHERE
			sm.shelf_id = :shelf_id
	`
	args := map[string]any{
		"shelf_id": req.ShelfId,
	}

	rows, err := hsqlx.SelectNamedCtx[DBBookMetadata](ctx, p.db, stmt, args)
	if err != nil {
		return nil, fmt.Errorf("error selecting: %w", err)
	}

	return hlp.Map(rows, func(r DBBookMetadata, _ int) *pbv1.Metadata {
		return DBBookMetadataToPBMetadata(r)
	}), nil
}
