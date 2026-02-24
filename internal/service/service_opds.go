package service

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/pauth"
	"github.com/nicjohnson145/blankpage/internal/storage"
	"github.com/nicjohnson145/hlp"
	"github.com/oklog/ulid/v2"
)

type xmlFunc func(req *http.Request) (any, error)

func (s *Service) respondWithXML(w http.ResponseWriter, req *http.Request, inner xmlFunc) {
	val, err := inner(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := xml.MarshalIndent(val, "", "   ")
	if err != nil {
		http.Error(w, "error marshalling response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func (s *Service) OPDSRoot(w http.ResponseWriter, req *http.Request) {
	if err := pauth.EnsureSessionHasOneOfRole(req.Context(), RoleViewOnly); err != nil {
		http.Error(w, "error ensuring access", http.StatusUnauthorized)
		return
	}
	s.respondWithXML(w, req, func(req *http.Request) (any, error) {
		return Feed{
			NS: FeedNamespace,
			// TODO: i think this is cached somehow? need to read the spec
			ID: ulid.Make().String(),
			// TODO: this should reflect data update time
			Updated: s.nowFunc(),
			Links: []FeedLink{
				{
					Rel:  "self",
					Href: RouteRoot(),
					Type: NavigationLinkKind,
				},
			},
			Entries: []FeedEntry{
				{
					Title: "All",
					Links: []FeedLink{
						{
							Href: routeFilterAllAlphabetical(),
							Type: CatalogLinkKind,
						},
					},
				},
				{
					Title: "Recently Added",
					Links: []FeedLink{
						{
							Href: routeFilterRecentlyAdded(),
							Type: CatalogLinkKind,
						},
					},
				},
				{
					Title: "Series",
					Links: []FeedLink{
						{
							Href: routeFilterSeries(),
							Type: CatalogLinkKind,
						},
					},
				},
			},
		}, nil
	})
}

func (s *Service) OPDSFilter(w http.ResponseWriter, req *http.Request) {
	if err := pauth.EnsureSessionHasOneOfRole(req.Context(), RoleViewOnly); err != nil {
		http.Error(w, "error ensuring access", http.StatusUnauthorized)
		return
	}
	s.respondWithXML(w, req, func(req *http.Request) (any, error) {
		filterKind := req.PathValue("kind")
		switch filterKind {
		case routeFilterKindSeries():
			return s.opdsFilterSeries(req)
		case routeFilterKindRecentlyAdded():
			return s.opdsFilterRecentlyAdded(req)
		case routeFilterKindAllAlphabetical():
			return s.opdsFilterAlphabetical(req)
		default:
			err := errors.New("handled filter kind " + filterKind)
			return nil, s.logError(req.Context(), err, "")
		}
	})
}

func (s *Service) opdsFilterSeries(req *http.Request) (*Feed, error) {
	series, err := s.storer.GetAllSeries(req.Context())
	if err != nil {
		return nil, s.logError(req.Context(), err, "error listing series")
	}

	entries := hlp.Map(series, func(str string, _ int) FeedEntry {
		return FeedEntry{
			Title: str,
			Links: []FeedLink{
				{
					Rel:  "subsection",
					Type: CatalogLinkKind,
					Href: routeSingleSeries(str),
				},
			},
		}
	})

	feed := &Feed{
		NS: FeedNamespace,
		// TODO: i think this is cached somehow? need to read the spec
		ID: ulid.Make().String(),
		// TODO: this should reflect data update time
		Updated: s.nowFunc(),
		Links: []FeedLink{
			{
				Rel:  "self",
				Href: RouteRoot(),
				Type: NavigationLinkKind,
			},
			{
				Rel:  "start",
				Href: RouteRoot(),
				Type: NavigationLinkKind,
			},
			{
				Rel:  "up",
				Href: RouteRoot(),
				Type: NavigationLinkKind,
			},
		},
		Entries: entries,
	}
	return feed, nil
}

func (s *Service) opdsFilterRecentlyAdded(req *http.Request) (*Feed, error) {
	storeResp, err := s.storer.ListMetadata(req.Context(), &pbv1.ListBooksRequest{
		PaginationOptions: &pbv1.ListBooksRequest_PaginationOptions{
			PerPage: hlp.Ptr(uint32(25)),
			Page:    hlp.Ptr(uint32(0)),
		},
		SortOptions: &pbv1.ListBooksRequest_SortOptions{
			SortField: pbv1.ListBooksRequest_SortOptions_SORT_FIELD_UPLOAD_DATE,
			SortOrder: pbv1.ListBooksRequest_SortOptions_SORT_ORDER_DESCENDING,
		},
	})
	if err != nil {
		return nil, s.logError(req.Context(), err, "error listing")
	}

	return &Feed{
		NS: FeedNamespace,
		// TODO: i think this is cached somehow? need to read the spec
		ID: ulid.Make().String(),
		// TODO: this should reflect data update time
		Updated: s.nowFunc(),
		Links: []FeedLink{
			{
				Rel:  "self",
				Href: routeFilterRecentlyAdded(),
				Type: NavigationLinkKind,
			},
			{
				Rel:  "start",
				Href: RouteRoot(),
				Type: NavigationLinkKind,
			},
			{
				Rel:  "up",
				Href: RouteRoot(),
				Type: NavigationLinkKind,
			},
		},
		Entries: hlp.Map(storeResp.Books, func(meta *pbv1.Metadata, _ int) FeedEntry {
			return metadataToFeedEntry(meta)
		}),
	}, nil
}

func (s *Service) opdsFilterAlphabetical(req *http.Request) (*Feed, error) {
	const pageSize uint32 = 30
	var page uint32 = 0
	pageStr := req.URL.Query().Get("page")
	if pageStr != "" {
		pageVal, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, s.logError(req.Context(), err, "error converting page query param")
		}
		page = uint32(pageVal)
	}

	storeResp, err := s.storer.ListMetadata(req.Context(), &pbv1.ListBooksRequest{
		PaginationOptions: &pbv1.ListBooksRequest_PaginationOptions{
			PerPage: hlp.Ptr(pageSize),
			Page:    hlp.Ptr(page),
		},
		SortOptions: &pbv1.ListBooksRequest_SortOptions{
			SortField: pbv1.ListBooksRequest_SortOptions_SORT_FIELD_TITLE,
			SortOrder: pbv1.ListBooksRequest_SortOptions_SORT_ORDER_ASCENDING,
		},
	})
	if err != nil {
		return nil, s.logError(req.Context(), err, "error listing")
	}

	links := []FeedLink{
		{
			Rel:  "self",
			Href: routeFilterAllAlphabetical(),
			Type: NavigationLinkKind,
		},
		{
			Rel:  "start",
			Href: RouteRoot(),
			Type: NavigationLinkKind,
		},
		{
			Rel:  "up",
			Href: RouteRoot(),
			Type: NavigationLinkKind,
		},
	}
	if storeResp.HasMore {
		links = append(links, FeedLink{
			Rel:  "next",
			Href: routeFilterAllAlphabetical() + "?page=" + fmt.Sprint(page+1),
			Type: NavigationLinkKind,
		})
	}
	if page != 0 {
		links = append(links, FeedLink{
			Rel:  "previous",
			Href: routeFilterAllAlphabetical() + "?page=" + fmt.Sprint(page-1),
			Type: NavigationLinkKind,
		})
	}

	return &Feed{
		NS: FeedNamespace,
		// TODO: i think this is cached somehow? need to read the spec
		ID: ulid.Make().String(),
		// TODO: this should reflect data update time
		Updated: s.nowFunc(),
		Links:   links,
		Entries: hlp.Map(storeResp.Books, func(meta *pbv1.Metadata, _ int) FeedEntry {
			return metadataToFeedEntry(meta)
		}),
	}, nil
}

func (s *Service) OPDSSingleSeries(w http.ResponseWriter, req *http.Request) {
	s.respondWithXML(w, req, func(req *http.Request) (any, error) {
		seriesName := req.PathValue("series_name")
		books, err := s.storer.GetBooksBySeries(req.Context(), seriesName)
		if err != nil {
			return nil, s.logError(req.Context(), err, "error getting by name")
		}

		entries := hlp.Map(books, func(meta *pbv1.Metadata, _ int) FeedEntry {
			return metadataToFeedEntry(meta)
		})

		feed := &Feed{
			NS: FeedNamespace,
			// TODO: i think this is cached somehow? need to read the spec
			ID: ulid.Make().String(),
			// TODO: this should reflect data update time
			Updated: s.nowFunc(),
			Links: []FeedLink{
				{
					Rel:  "self",
					Href: routeSingleSeries(seriesName),
					Type: NavigationLinkKind,
				},
				{
					Rel:  "start",
					Href: RouteRoot(),
					Type: NavigationLinkKind,
				},
				{
					Rel:  "up",
					Href: routeFilterSeries(),
					Type: NavigationLinkKind,
				},
			},
			Entries: entries,
		}
		return feed, nil
	})
}

func (s *Service) OPDSDownload(w http.ResponseWriter, req *http.Request) {
	if err := pauth.EnsureSessionHasOneOfRole(req.Context(), RoleViewOnly); err != nil {
		http.Error(w, "error ensuring access", http.StatusUnauthorized)
		return
	}

	book, err := s.storer.ReadBook(req.Context(), req.PathValue("id"))
	if err != nil {
		if errors.Is(err, storage.ErrNotFoundError) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}
	}

	// TODO: format check whats available on a per-book basis
	if req.PathValue("format") != "epub" {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte("format not available for this book"))
		return
	}

	w.WriteHeader(http.StatusOK)
	io.Copy(w, bytes.NewReader(book.Content)) //nolint: errcheck
}
