package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	pbv1beta1 "github.com/nicjohnson145/blankpage/gen/go/pauth/v1beta1"
	"github.com/nicjohnson145/blankpage/internal/svcconfig"
	"github.com/spf13/viper"
)

var (
	ErrUserAlreadyExistsError        = errors.New("a user with this id already exists")
	ErrUnknownUserError              = errors.New("unknown user")
	ErrIncorrectPasswordError        = errors.New("password does not match")
	ErrNoPasswordConfiguredError     = errors.New("no password configured")
	ErrSessionUnknownOrInactiveError = errors.New("session unknwon or inactive")
)

type CreateUserRequest struct {
	User           *pbv1beta1.User
	HashedPassword *string
}

type SetUserPasswordRequest struct {
	UserId         string
	HashedPassword string
}

type CreateSessionRequest struct {
	Session     *Session
	ActiveUntil time.Time
}

type LoginInfo struct {
	User           *pbv1beta1.User
	StoredPassword string
}

type Storer interface {
	Purge(ctx context.Context) error

	CreateUser(ctx context.Context, req *CreateUserRequest) error
	ReadUser(ctx context.Context, userId string) (*pbv1beta1.User, error)
	ReadLoginInfo(ctx context.Context, userID *string, email *string) (*LoginInfo, error)
	ListUsers(ctx context.Context, req *pbv1beta1.ListUsersRequest) ([]*pbv1beta1.User, error)
	SetUserPassword(ctx context.Context, req *SetUserPasswordRequest) error
	UpdateUser(ctx context.Context, user *pbv1beta1.User) error
	DeleteUser(ctx context.Context, req *pbv1beta1.DeleteUserRequest) error
	ListUserRoles(ctx context.Context, userId string) ([]string, error)
	GrantUserRoles(ctx context.Context, req *pbv1beta1.GrantUserRoleRequest) error
	RevokeUserRoles(ctx context.Context, req *pbv1beta1.RevokeUserRoleRequest) error

	GetActiveSession(ctx context.Context, sessionKey string) (*Session, error)
	CreateSession(ctx context.Context, req *CreateSessionRequest) error
	RemoveSession(ctx context.Context, sessionKey string) error
}

func NewFromEnv() (Storer, func(), error) {
	kind, err := svcconfig.ParseStorageKind(viper.GetString(svcconfig.StorageType))
	if err != nil {
		return nil, func() {}, err
	}

	switch kind {
	case svcconfig.StorageKindMemory:
		return NewMemory(MemoryConfig{}), func() {}, nil
	default:
		return nil, func() {}, fmt.Errorf("unhandled storage kind %v", kind)
	}
}
