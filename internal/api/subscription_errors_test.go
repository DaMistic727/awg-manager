package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hoaxisr/awg-manager/internal/singbox/subscription"
	sysfiles "github.com/hoaxisr/awg-manager/internal/sys/files"
)

// Одна таблица и для кода ответа, и для признака «внутренний сбой» (F430):
// каждый сентинел — свой 4xx, прочее — 500 без кода.
func TestSubscriptionErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("x: %w", &subscription.FilterError{}), http.StatusBadRequest, "INVALID_FILTER"},
		{fmt.Errorf("x: %w", subscription.ErrInvalidInput), http.StatusBadRequest, "INVALID_INPUT"},
		{fmt.Errorf("x: %w", subscription.ErrValidation), http.StatusUnprocessableEntity, "VALIDATION_FAILED"},
		{fmt.Errorf("x: %w", subscription.ErrExcludeOnInline), http.StatusBadRequest, "EXCLUDE_ON_INLINE"},
		{fmt.Errorf("x: %w", subscription.ErrAllMembersExcluded), http.StatusConflict, "ALL_MEMBERS_EXCLUDED"},
		{fmt.Errorf("x: %w", subscription.ErrAllMembersFiltered), http.StatusConflict, "ALL_MEMBERS_FILTERED"},
		{fmt.Errorf("x: %w", subscription.ErrMemberNotFound), http.StatusNotFound, "MEMBER_NOT_FOUND"},
		{fmt.Errorf("x: %w", sysfiles.ErrPathDenied), http.StatusBadRequest, "PATH_DENIED"},
		{errors.New("disk full"), http.StatusInternalServerError, ""},
	} {
		if s, c := subscriptionErrorStatus(tc.err); s != tc.status || c != tc.code {
			t.Errorf("%v: %d %q, want %d %q", tc.err, s, c, tc.status, tc.code)
		}
	}
}
