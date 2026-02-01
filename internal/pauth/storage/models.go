package storage

import pbv1beta1 "github.com/nicjohnson145/blankpage/gen/go/pauth/v1beta1"

type Session struct {
	ID   string
	User *pbv1beta1.User
}
