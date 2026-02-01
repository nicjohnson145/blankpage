package service

import (
	"testing"

	"buf.build/go/protovalidate"
	pbv1 "github.com/nicjohnson145/blankpage/gen/go/blankpage/v1"
	"github.com/stretchr/testify/require"
)

func TestAddBookRequestValidation(t *testing.T) {
	t.Parallel()

	valid := func() *pbv1.AddBookRequest {
		return &pbv1.AddBookRequest{
			Book: &pbv1.Book{
				Metadata: &pbv1.Metadata{
					Title: "abc",
				},
				Content: []byte("def"),
			},
		}
	}

	testData := []struct {
		name        string
		transform   func(x *pbv1.AddBookRequest)
		expectedErr string
	}{
		{
			name:        "valid",
			transform:   func(x *pbv1.AddBookRequest) {},
			expectedErr: "",
		},
		{
			name: "no content",
			transform: func(x *pbv1.AddBookRequest) {
				x.Book.Content = nil
			},
			expectedErr: "content: value is required",
		},
		{
			name: "no title",
			transform: func(x *pbv1.AddBookRequest) {
				x.Book.Metadata.Title = ""
			},
			expectedErr: "title: value is required",
		},
		{
			name: "no metadata",
			transform: func(x *pbv1.AddBookRequest) {
				x.Book.Metadata = nil
			},
			expectedErr: "metadata: value is required",
		},
	}

	for _, tc := range testData {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := valid()
			tc.transform(v)

			err := protovalidate.Validate(v)
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr)
			}
		})
	}
}

func TestUpdateBookRequestValidation(t *testing.T) {
	t.Parallel()

	valid := func() *pbv1.UpdateBookRequest {
		return &pbv1.UpdateBookRequest{
			Book: &pbv1.Book{
				Metadata: &pbv1.Metadata{
					Id:    "abc",
					Title: "def",
				},
			},
		}
	}

	testData := []struct {
		name        string
		transform   func(x *pbv1.UpdateBookRequest)
		expectedErr string
	}{
		{
			name:        "valid",
			transform:   func(x *pbv1.UpdateBookRequest) {},
			expectedErr: "",
		},
		{
			name: "no id",
			transform: func(x *pbv1.UpdateBookRequest) {
				x.Book.Metadata.Id = ""
			},
			expectedErr: "id: value is required",
		},
	}

	for _, tc := range testData {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := valid()
			tc.transform(v)

			err := protovalidate.Validate(v)
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr)
			}
		})
	}
}
