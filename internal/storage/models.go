package storage

import (
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/nicjohnson145/hlp/set"
)

type memoryShelf struct {
	Shelf *pbv1.Shelf
	Books *set.Set[string]
}
