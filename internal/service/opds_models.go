package service

import (
	"encoding/xml"
	"time"

	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
)

const (
	FeedNamespace      = "http://www.w3.org/2005/Atom"
	NavigationLinkKind = "application/atom+xml;profile=opds-catalog;kind=navigation"
	CatalogLinkKind    = "application/atom+xml;profile=opds-catalog"
	EpubLinkType       = "application/epub+zip"
)

type Feed struct {
	XMLName xml.Name    `xml:"feed"`
	NS      string      `xml:"ns,attr"`
	ID      string      `xml:"id"`
	Updated time.Time   `xml:"updated"`
	Links   []FeedLink  `xml:"link"`
	Entries []FeedEntry `xml:"entry"`
}

type FeedLink struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Href    string   `xml:"href,attr"`
	Type    string   `xml:"type,attr"`
	Title   string   `xml:"title,attr,omitempty"`
}

type FeedEntry struct {
	XMLName xml.Name    `xml:"entry"`
	ID      string      `xml:"id,omitempty"`
	Title   string      `xml:"title"`
	Links   []FeedLink  `xml:"link"`
	Author  *FeedAuthor `xml:"author,omitempty"`
}

type FeedAuthor struct {
	Name string `xml:"name"`
}

func metadataToFeedEntry(metadata *pbv1.Metadata) FeedEntry {
	entry := FeedEntry{
		ID:    metadata.Id,
		Title: metadata.Title,
		Links: []FeedLink{
			{
				Rel:   "http://opds-spec.org/acquisition",
				Href:  routeDownloadEpub(metadata.Id),
				Title: "EPUB",
				Type:  EpubLinkType,
			},
			// TODO: cover images and what not
		},
	}
	if metadata.Author != nil {
		entry.Author = &FeedAuthor{
			Name: *metadata.Author,
		}
	}

	return entry
}
