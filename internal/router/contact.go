package router

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us *UserService) PatchContact(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	contact := &shared.Contact{}
	ctx := r.Context()

	userid := shared.UUID(chi.URLParam(r, "user_id"))
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
	} else if err = json.Unmarshal(body, contact); err != nil {
		sc(http.StatusBadRequest).send(ctx, w, err)
		_, _ = w.Write([]byte(fmt.Sprintf("couldn't unmarshal: '%s'", html.EscapeString(string(body)))))
	} else if err = us.Contacter.UpdateContact(ctx, userid, contact); err != nil {
		sc(http.StatusInternalServerError).send(ctx, w, err, err.Error())
	} else {
		sc(http.StatusOK).success(ctx, w)
	}
}
