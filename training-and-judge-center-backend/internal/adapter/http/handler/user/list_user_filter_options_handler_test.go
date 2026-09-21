package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithListUserFilterOptions(uc *appuser.ListUserFilterOptionsUseCase) *Handler {
	return &Handler{listUserFilterOptions: uc}
}

func TestListUserFilterOptions_Authenticated_Returns200(t *testing.T) {
	repo := &mockHandlerUserRepo{
		findFilterOptionsFn: func(_ context.Context) (domainuser.FilterOptions, error) {
			return domainuser.FilterOptions{
				Countries:    []string{"Colombia", "Mexico"},
				Cities:       []string{"Bogota", "Cucuta"},
				Institutions: []string{"UFPS"},
			}, nil
		},
	}
	uc := appuser.NewListUserFilterOptionsUseCase(repo)
	h := newHandlerWithListUserFilterOptions(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.ListUserFilterOptions),
		&domainuser.TokenClaims{UserID: "admin-1", Role: shared.RoleAdmin},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/users/filters", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp listUserFilterOptionsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.Countries) != 2 || resp.Countries[0] != "Colombia" {
		t.Errorf("unexpected countries: %+v", resp.Countries)
	}
	if len(resp.Cities) != 2 {
		t.Errorf("unexpected cities: %+v", resp.Cities)
	}
	if len(resp.Institutions) != 1 || resp.Institutions[0] != "UFPS" {
		t.Errorf("unexpected institutions: %+v", resp.Institutions)
	}
}
