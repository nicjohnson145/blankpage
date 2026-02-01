package storage

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"

	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/hlp"
	"github.com/nicjohnson145/hlp/set"
	"google.golang.org/protobuf/proto"
)

type MemoryConfig struct{}

func NewMemory(conf MemoryConfig) *Memory {
	return &Memory{
		mu:      &sync.RWMutex{},
		books:   map[string]*pbv1.Book{},
		shelves: map[string]memoryShelf{},
	}
}

type Memory struct {
	mu      *sync.RWMutex
	books   map[string]*pbv1.Book
	shelves map[string]memoryShelf
}

func (m *Memory) Purge(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.books = map[string]*pbv1.Book{}
	m.shelves = map[string]memoryShelf{}

	return nil
}

func (m *Memory) AddBook(ctx context.Context, req *pbv1.AddBookRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.books[req.Book.Metadata.Id] = req.Book

	return nil
}

func (m *Memory) ReadBook(ctx context.Context, id string) (*pbv1.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	book, ok := m.books[id]
	if !ok {
		return nil, ErrNotFoundError
	}

	return book, nil
}

func (m *Memory) RemoveBook(ctx context.Context, req *pbv1.RemoveBookRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove the actual book
	delete(m.books, req.BookId)

	// Remove the book from any shelves its on
	for id := range m.shelves {
		m.shelves[id].Books.Remove(req.BookId)
	}

	return nil
}

func (m *Memory) ListMetadata(ctx context.Context, req *pbv1.ListBooksRequest) ([]*pbv1.Metadata, uint32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get all the books
	books := hlp.Values(m.books)

	type sortFunc func(a *pbv1.Book, b *pbv1.Book) int

	// Sort them by the requested field
	var fieldSortFunc sortFunc
	switch req.SortOptions.SortField {
	case pbv1.ListBooksRequest_SortOptions_SORT_FIELD_UPLOAD_DATE:
		fieldSortFunc = func(a, b *pbv1.Book) int {
			return a.Metadata.UploadedAt.AsTime().Compare(b.Metadata.UploadedAt.AsTime())
		}
	case pbv1.ListBooksRequest_SortOptions_SORT_FIELD_TITLE:
		fieldSortFunc = func(a, b *pbv1.Book) int {
			return strings.Compare(a.Metadata.Title, b.Metadata.Title)
		}
	default:
		return nil, 0, fmt.Errorf("unhandled sort field %v", req.SortOptions.SortField)
	}

	var finalSortFunc sortFunc
	switch req.SortOptions.SortOrder {
	case pbv1.ListBooksRequest_SortOptions_SORT_ORDER_DESCENDING:
		finalSortFunc = func(a, b *pbv1.Book) int {
			return fieldSortFunc(a, b) * -1
		}
	case pbv1.ListBooksRequest_SortOptions_SORT_ORDER_ASCENDING:
		finalSortFunc = fieldSortFunc
	}

	slices.SortFunc(books, finalSortFunc)

	// Get the bounds
	lower := int(*req.PaginationOptions.Page * *req.PaginationOptions.PerPage)
	upper := int(math.Min(float64(lower+int(*req.PaginationOptions.PerPage)), float64(len(books))))

	// If we're asking for a page that doesnt exist, just return nothing
	if int(lower) > len(books) {
		return []*pbv1.Metadata{}, 0, nil
	}

	// Otherwise, for every item in books starting at the lower and going to the upper, copy it out
	outBooks := []*pbv1.Metadata{}
	for idx := lower; idx < upper; idx++ {
		outBooks = append(outBooks, proto.Clone(books[idx].Metadata).(*pbv1.Metadata))
	}

	return outBooks, uint32(len(books)), nil
}

func (m *Memory) GetAllSeries(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	series := set.New[string]()
	for _, book := range m.books {
		if book.Metadata.Series == nil {
			continue
		}
		series.Add(*book.Metadata.Series)
	}

	seriesSlice := series.AsSlice()
	slices.Sort(seriesSlice)

	return seriesSlice, nil
}

func (m *Memory) GetBooksBySeries(ctx context.Context, series string) ([]*pbv1.Metadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	books := []*pbv1.Metadata{}
	allHaveSeriesNum := true
	for _, book := range m.books {
		if book.Metadata.Series == nil || *book.Metadata.Series != series {
			continue
		}

		books = append(books, book.Metadata)
		if book.Metadata.SeriesNumber == nil {
			allHaveSeriesNum = false
		}
	}

	// If all the books in the series have a number, then we can sort by that, otherwise just do alphabetical
	if allHaveSeriesNum {
		slices.SortFunc(books, func(a *pbv1.Metadata, b *pbv1.Metadata) int {
			val := *a.SeriesNumber - *b.SeriesNumber
			if val < 0 {
				return -1
			}
			if val == 0 {
				return 0
			}
			return 1
		})
	} else {
		slices.SortFunc(books, func(a *pbv1.Metadata, b *pbv1.Metadata) int {
			return strings.Compare(a.Title, b.Title)
		})
	}

	return books, nil
}

func (m *Memory) CreateShelf(ctx context.Context, shelf *pbv1.Shelf) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shelves[shelf.Id] = memoryShelf{
		Shelf: shelf,
		Books: set.New[string](),
	}

	return nil
}

func (m *Memory) ListShelves(ctx context.Context) ([]*pbv1.Shelf, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shelves := hlp.Map(hlp.Values(m.shelves), func(shelf memoryShelf, _ int) *pbv1.Shelf {
		return shelf.Shelf
	})
	slices.SortFunc(shelves, func(a *pbv1.Shelf, b *pbv1.Shelf) int {
		return -1 * strings.Compare(a.Id, b.Id)
	})
	return shelves, nil
}

func (m *Memory) AddBooksToShelf(ctx context.Context, req *pbv1.AddBooksToShelfRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	shelf, ok := m.shelves[req.ShelfId]
	if !ok {
		return ErrUnknownShelfError
	}

	for _, id := range req.BookIds {
		if _, ok := m.books[id]; !ok {
			return fmt.Errorf("book %v: %w", id, ErrUnknownBookError)
		}
		shelf.Books.Add(id)
	}

	m.shelves[req.ShelfId] = shelf

	return nil
}

func (m *Memory) RemoveBooksFromShelf(ctx context.Context, req *pbv1.RemoveBooksFromShelfRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	shelf, ok := m.shelves[req.ShelfId]
	if !ok {
		return ErrUnknownShelfError
	}

	for _, id := range req.BookIds {
		shelf.Books.Remove(id)
	}
	m.shelves[req.ShelfId] = shelf

	return nil
}

func (m *Memory) ListBooksForShelf(ctx context.Context, req *pbv1.ListBooksForShelfRequest) ([]*pbv1.Metadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shelf, ok := m.shelves[req.ShelfId]
	if !ok {
		return nil, ErrUnknownShelfError
	}

	books := []*pbv1.Metadata{}
	for _, bookId := range shelf.Books.Iter() {
		books = append(books, m.books[bookId].Metadata)
	}

	return books, nil
}
