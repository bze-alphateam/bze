package types

import (
	"strings"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgAddArticle(t *testing.T) {
	publisher := sample.AccAddress()
	title := "Test Article Title"
	url := "https://example.com"

	msg := NewMsgAddArticle(publisher, title, url)

	require.Equal(t, publisher, msg.Publisher)
	require.Equal(t, title, msg.Title)
	require.Equal(t, url, msg.Url)
}

func TestMsgAddArticle_ValidateBasic(t *testing.T) {
	validPublisher := sample.AccAddress()
	validTitle := "Valid Article Title"
	validUrl := "https://example.com/article"
	validPicture := "https://example.com/image.jpg"

	// Create strings for length testing
	shortTitle := "Short"                                         // 5 chars, below titleMinLength (10)
	longTitle := strings.Repeat("a", 321)                         // 321 chars, above titleMaxLength (320)
	longUrl := "https://example.com/" + strings.Repeat("a", 2030) // ~2048+ chars, above urlMaxLength (2048)

	tests := []struct {
		name string
		msg  MsgAddArticle
		err  error
	}{
		{
			name: "invalid publisher address",
			msg: MsgAddArticle{
				Publisher: "invalid_address",
				Title:     validTitle,
				Url:       validUrl,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty publisher address",
			msg: MsgAddArticle{
				Publisher: "",
				Title:     validTitle,
				Url:       validUrl,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "title too short",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     shortTitle,
				Url:       validUrl,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "title too long",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     longTitle,
				Url:       validUrl,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "empty title",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     "",
				Url:       validUrl,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid url - empty",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid url - malformed",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "not-a-url",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid url - http scheme",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "http://example.com",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid url - ftp scheme",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "ftp://example.com",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid url - no scheme",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "example.com",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "url too long",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       longUrl,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid picture url - malformed",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   "not-a-url",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid picture url - http scheme",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   "http://example.com/image.jpg",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid picture url - no scheme",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   "example.com/image.jpg",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "picture url too long",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   longUrl,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - no picture",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   "",
			},
		},
		{
			name: "valid message - with picture",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       validUrl,
				Picture:   validPicture,
			},
		},
		{
			name: "valid message - https url with path",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "https://example.com/path/to/article?param=value",
				Picture:   validPicture,
			},
		},
		{
			name: "valid message - https url with port",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "https://example.com:8080/article",
				Picture:   "https://example.com:8080/image.png",
			},
		},
		{
			name: "valid message - subdomain",
			msg: MsgAddArticle{
				Publisher: validPublisher,
				Title:     validTitle,
				Url:       "https://blog.example.com/article",
				Picture:   "https://cdn.example.com/image.jpg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgAddArticle_ParseUrl(t *testing.T) {
	msg := &MsgAddArticle{}

	tests := []struct {
		name      string
		url       string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid https url",
			url:       "https://example.com",
			shouldErr: false,
		},
		{
			name:      "valid https url with path",
			url:       "https://example.com/path/to/resource",
			shouldErr: false,
		},
		{
			name:      "valid https url with query params",
			url:       "https://example.com/path?param=value",
			shouldErr: false,
		},
		{
			name:      "valid https url with port",
			url:       "https://example.com:8080/path",
			shouldErr: false,
		},
		{
			name:      "invalid url - malformed",
			url:       "not-a-url",
			shouldErr: true,
		},
		{
			name:      "invalid url - http scheme",
			url:       "http://example.com",
			shouldErr: true,
			errMsg:    "invalid url scheme: only https accepted",
		},
		{
			name:      "invalid url - ftp scheme",
			url:       "ftp://example.com",
			shouldErr: true,
			errMsg:    "invalid url scheme: only https accepted",
		},
		{
			name:      "invalid url - no scheme",
			url:       "example.com",
			shouldErr: true,
		},
		{
			name:      "invalid url - empty",
			url:       "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := msg.ParseUrl(tt.url)
			if tt.shouldErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, parsed)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, parsed)
			require.Equal(t, "https", parsed.Scheme)
		})
	}
}
