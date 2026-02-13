package storage

import (
	"database/sql"
	"time"

	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/hlp"
	"github.com/nicjohnson145/hlp/set"
	hsqlx "github.com/nicjohnson145/hlp/sqlx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type memoryShelf struct {
	Shelf *pbv1.Shelf
	Books *set.Set[string]
}

type DBBookMetadata struct {
	ID           string            `db:"id"`
	Title        string            `db:"title"`
	Author       sql.Null[string]  `db:"author"`
	Series       sql.Null[string]  `db:"series"`
	SeriesNumber sql.Null[float32] `db:"series_number"`
	UploadTS     time.Time         `db:"upload_ts"`
}

func PBMetadataToDBBookMetadata(metadata *pbv1.Metadata) DBBookMetadata {
	return DBBookMetadata{
		ID:           metadata.Id,
		Title:        metadata.Title,
		Author:       hsqlx.PointerToSqlNull(metadata.Author),
		Series:       hsqlx.PointerToSqlNull(metadata.Series),
		SeriesNumber: hsqlx.PointerToSqlNull(metadata.SeriesNumber),
		UploadTS:     metadata.UploadedAt.AsTime(),
	}
}

func DBBookMetadataToPBMetadata(row DBBookMetadata) *pbv1.Metadata {
	return &pbv1.Metadata{
		Id:           row.ID,
		Title:        row.Title,
		Author:       hsqlx.SqlNullToPointer(row.Author),
		Series:       hsqlx.SqlNullToPointer(row.Series),
		SeriesNumber: hsqlx.SqlNullToPointer(row.SeriesNumber),
		UploadedAt:   timestamppb.New(row.UploadTS),
	}
}

type DBBookTag struct {
	BookID string `db:"book_id"`
	Tag    string `db:"tag"`
}

type DBBookContent struct {
	BookID  string `db:"book_id"`
	Format  string `db:"format"`
	Content []byte `db:"content"`
}

type DBShelf struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func PBShelfToDBShelf(shelf *pbv1.Shelf) DBShelf {
	return DBShelf{
		ID:   shelf.Id,
		Name: shelf.Name,
	}
}

func DBShelfToPBShelf(row DBShelf) *pbv1.Shelf {
	return &pbv1.Shelf{
		Id:   row.ID,
		Name: row.Name,
	}
}

type DBShelfMember struct {
	ShelfID string `db:"shelf_id"`
	BookID  string `db:"book_id"`
}

func PBAddShelfBooksRequestToDBShelfMembers(req *pbv1.AddBooksToShelfRequest) []DBShelfMember {
	return hlp.Map(req.BookIds, func(id string, _ int) DBShelfMember {
		return DBShelfMember{
			ShelfID: req.ShelfId,
			BookID:  id,
		}
	})
}

type DBOutputBook struct {
	DBBookMetadata
	Content []byte `db:"content"`
}

func DBOutputBookToPBBook(row DBOutputBook) *pbv1.Book {
	return &pbv1.Book{
		Metadata: DBBookMetadataToPBMetadata(row.DBBookMetadata),
		Content:  row.Content,
	}
}
