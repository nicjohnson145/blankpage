package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/timsims/pamphlet"

	"connectrpc.com/connect"
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	pbv1connect "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1/blankpagev1connect"
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/nicjohnson145/pauth"
	"github.com/nicjohnson145/blankpage/internal/storage"
	"github.com/nicjohnson145/hlp"
	"github.com/oklog/ulid/v2"
	"go.einride.tech/aip/fieldmask"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrEndpointDisabledError = errors.New("endpoint disabled")
)

const (
	RoleAdmin        = pauth.RoleAdministrator
	RoleBookUploader = "book-uploader"
	RoleBookUpdater  = "book-updater"
	RoleShelfAdmin   = "shelf-admin"
	RoleViewOnly     = "view-only"
)

type ServiceConfig struct {
	Storer storage.Storer
	// PurgeEnabled enables the purge endpoint. This endpoint should never be enabled on a non-testing deployment
	PurgeEnabled   bool
	DefaultPerPage uint32
	NowFunc        func() time.Time
}

func NewService(conf ServiceConfig) *Service {
	now := conf.NowFunc
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &Service{
		storer:         conf.Storer,
		purgeEnabled:   conf.PurgeEnabled,
		defaultPerPage: conf.DefaultPerPage,
		nowFunc:        now,
	}
}

type Service struct {
	pbv1connect.UnimplementedBlankPageServiceHandler

	storer         storage.Storer
	purgeEnabled   bool
	defaultPerPage uint32
	nowFunc        func() time.Time
}

func (s *Service) logError(ctx context.Context, err error, msg string) error {

	log := logging.GetContextLogger(ctx)

	outMsg := msg
	if outMsg == "" {
		outMsg = "an error occurred"
	}

	log.Err(err).Msg(outMsg)

	switch true {
	case errors.Is(err, storage.ErrUnknownShelfError):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, storage.ErrUnknownBookError):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return err
	}
}

func (s *Service) Purge(ctx context.Context, req *connect.Request[pbv1.PurgeRequest]) (*connect.Response[pbv1.PurgeResponse], error) {
	if !s.purgeEnabled {
		return nil, s.logError(ctx, ErrEndpointDisabledError, "purge is disabled")
	}

	// Purge everything
	if err := s.storer.Purge(ctx); err != nil {
		return nil, s.logError(ctx, err, "error purging")
	}

	return connect.NewResponse(&pbv1.PurgeResponse{}), nil
}

func (s *Service) AddBook(ctx context.Context, req *connect.Request[pbv1.AddBookRequest]) (*connect.Response[pbv1.AddBookResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleBookUploader); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	// Default the ID if not given
	if req.Msg.Book.Metadata.Id == "" {
		req.Msg.Book.Metadata.Id = ulid.Make().String()
	}

	// Set the uplaod time
	req.Msg.Book.Metadata.UploadedAt = timestamppb.New(s.nowFunc())

	// Add it to the backend
	if err := s.storer.AddBook(ctx, req.Msg); err != nil {
		return nil, s.logError(ctx, err, "error adding book")
	}

	return connect.NewResponse(&pbv1.AddBookResponse{
		Book: req.Msg.Book,
	}), nil
}

func (s *Service) ReadBook(ctx context.Context, req *connect.Request[pbv1.ReadBookRequest]) (*connect.Response[pbv1.ReadBookResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleViewOnly); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	book, err := s.storer.ReadBook(ctx, req.Msg.BookId)
	if err != nil {
		return nil, s.logError(ctx, err, "error reading")
	}
	return connect.NewResponse(&pbv1.ReadBookResponse{
		Metadata: book.Metadata,
	}), nil
}

func (s *Service) UpdateBook(ctx context.Context, req *connect.Request[pbv1.UpdateBookRequest]) (*connect.Response[pbv1.UpdateBookResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleBookUpdater); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	book, err := s.storer.ReadBook(ctx, req.Msg.Book.Metadata.Id)
	if err != nil {
		return nil, s.logError(ctx, err, "error reading")
	}

	// Clone our original upload date, ensuring its preserved
	originalUploadTS := proto.Clone(book.Metadata.UploadedAt).(*timestamppb.Timestamp)

	updatedBook := req.Msg.Book

	if req.Msg.FieldMask != nil {
		fieldmask.Update(req.Msg.FieldMask, book, req.Msg.Book)
		updatedBook = book
	}

	// Replay that upload date back onto the new book
	updatedBook.Metadata.UploadedAt = originalUploadTS

	err = s.storer.AddBook(ctx, &pbv1.AddBookRequest{
		Book: updatedBook,
	})
	if err != nil {
		return nil, s.logError(ctx, err, "error updating")
	}

	return connect.NewResponse(&pbv1.UpdateBookResponse{}), nil
}

func (s *Service) RemoveBook(ctx context.Context, req *connect.Request[pbv1.RemoveBookRequest]) (*connect.Response[pbv1.RemoveBookResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleAdmin); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	if err := s.storer.RemoveBook(ctx, req.Msg); err != nil {
		return nil, s.logError(ctx, err, "error removing")
	}
	return connect.NewResponse(&pbv1.RemoveBookResponse{}), nil
}

func (s *Service) ListBooks(ctx context.Context, req *connect.Request[pbv1.ListBooksRequest]) (*connect.Response[pbv1.ListBooksResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleViewOnly); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	// normalize the list request, setting defaults
	s.normalizeListRequest(req.Msg)

	// Get our list
	storeResp, err := s.storer.ListMetadata(ctx, req.Msg)
	if err != nil {
		return nil, s.logError(ctx, err, "error listing")
	}

	return connect.NewResponse(&pbv1.ListBooksResponse{
		Books:   storeResp.Books,
		HasMore: storeResp.HasMore,
	}), nil
}

func (s *Service) normalizeListRequest(req *pbv1.ListBooksRequest) {
	if req.SortOptions == nil {
		req.SortOptions = &pbv1.ListBooksRequest_SortOptions{}
	}
	if req.PaginationOptions == nil {
		req.PaginationOptions = &pbv1.ListBooksRequest_PaginationOptions{}
	}

	if req.SortOptions.SortField == pbv1.ListBooksRequest_SortOptions_SORT_FIELD_UNSPECIFIED {
		req.SortOptions.SortField = pbv1.ListBooksRequest_SortOptions_SORT_FIELD_UPLOAD_DATE
	}
	if req.SortOptions.SortOrder == pbv1.ListBooksRequest_SortOptions_SORT_ORDER_UNSPECIFIED {
		req.SortOptions.SortOrder = pbv1.ListBooksRequest_SortOptions_SORT_ORDER_DESCENDING
	}

	if req.PaginationOptions.Page == nil {
		req.PaginationOptions.Page = hlp.Ptr(uint32(0))
	}
	if req.PaginationOptions.PerPage == nil {
		req.PaginationOptions.PerPage = hlp.Ptr(s.defaultPerPage)
	}
}

func (s *Service) ExtractMetadata(ctx context.Context, req *connect.Request[pbv1.ExtractMetadataRequest]) (*connect.Response[pbv1.ExtractMetadataResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleBookUploader); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	metadata, err := s.parseEpub(req.Msg)
	if err != nil {
		return nil, s.logError(ctx, err, "error parsing epub")
	}

	return connect.NewResponse(&pbv1.ExtractMetadataResponse{
		Metadata: metadata,
	}), nil
}

func (s *Service) parseEpub(req *pbv1.ExtractMetadataRequest) (*pbv1.Metadata, error) {
	parser, err := pamphlet.OpenBytes(req.Content)
	if err != nil {
		return nil, fmt.Errorf("error creating epub parser: %w", err)
	}
	defer func() {
		_ = parser.Close()
	}()

	book := parser.GetBook()

	// TODO: could probably try to pull more data out of this, but this will be a start
	metadata := &pbv1.Metadata{
		Title: book.Title,
	}
	if book.Author != "" {
		metadata.Author = hlp.Ptr(book.Author)
	}

	return metadata, nil
}

func (s *Service) CreateShelf(ctx context.Context, req *connect.Request[pbv1.CreateShelfRequest]) (*connect.Response[pbv1.CreateShelfResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleShelfAdmin); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	// Set our id if its not already
	if req.Msg.Shelf.Id == "" {
		req.Msg.Shelf.Id = ulid.Make().String()
	}

	if err := s.storer.CreateShelf(ctx, req.Msg.Shelf); err != nil {
		return nil, s.logError(ctx, err, "error creating")
	}

	return connect.NewResponse(&pbv1.CreateShelfResponse{
		Shelf: req.Msg.Shelf,
	}), nil
}

func (s *Service) ListShelves(ctx context.Context, req *connect.Request[pbv1.ListShelvesRequest]) (*connect.Response[pbv1.ListShelvesResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleViewOnly); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	shelves, err := s.storer.ListShelves(ctx)
	if err != nil {
		return nil, s.logError(ctx, err, "error listing")
	}

	return connect.NewResponse(&pbv1.ListShelvesResponse{
		Shelves: shelves,
	}), nil
}

func (s *Service) AddBooksToShelf(ctx context.Context, req *connect.Request[pbv1.AddBooksToShelfRequest]) (*connect.Response[pbv1.AddBooksToShelfResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleShelfAdmin); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	if err := s.storer.AddBooksToShelf(ctx, req.Msg); err != nil {
		return nil, s.logError(ctx, err, "error adding books")
	}

	return connect.NewResponse(&pbv1.AddBooksToShelfResponse{}), nil
}

func (s *Service) RemoveBooksFromShelf(ctx context.Context, req *connect.Request[pbv1.RemoveBooksFromShelfRequest]) (*connect.Response[pbv1.RemoveBooksFromShelfResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleShelfAdmin); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	if err := s.storer.RemoveBooksFromShelf(ctx, req.Msg); err != nil {
		return nil, s.logError(ctx, err, "error adding books")
	}

	return connect.NewResponse(&pbv1.RemoveBooksFromShelfResponse{}), nil
}

func (s *Service) ListBooksForShelf(ctx context.Context, req *connect.Request[pbv1.ListBooksForShelfRequest]) (*connect.Response[pbv1.ListBooksForShelfResponse], error) {
	if err := pauth.EnsureSessionHasOneOfRole(ctx, RoleViewOnly); err != nil {
		return nil, s.logError(ctx, err, "error ensuring access")
	}

	books, err := s.storer.ListBooksForShelf(ctx, req.Msg)
	if err != nil {
		return nil, s.logError(ctx, err, "error listing")
	}

	return connect.NewResponse(&pbv1.ListBooksForShelfResponse{
		Books: books,
	}), nil
}
