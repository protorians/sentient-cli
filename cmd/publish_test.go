package cmd

import (
	"net/http"
	"testing"

	"github.com/protorians/sentient-cli/internal/pkg"
)

func TestIsVersionConflict(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"409 conflict", &pkg.APIError{StatusCode: http.StatusConflict, Message: "version déjà publiée"}, true},
		{"message version existe", &pkg.APIError{StatusCode: http.StatusBadRequest, Message: "la version 0.1.0 existe déjà"}, true},
		{"message version conflict", &pkg.APIError{StatusCode: http.StatusBadRequest, Message: "version conflict detected"}, true},
		{"message sans version", &pkg.APIError{StatusCode: http.StatusBadRequest, Message: "manifest invalide"}, false},
		{"erreur non API", pkg.NewError("Publication", "boom", pkg.ExitPublish), false},
		{"erreur générique", &pkg.APIError{StatusCode: http.StatusBadRequest, Message: "bad request"}, false},
	}
	for _, tc := range cases {
		if got := isVersionConflict(tc.err); got != tc.want {
			t.Errorf("%s: isVersionConflict(%v) = %v, want %v", tc.name, tc.err, got, tc.want)
		}
	}
}
