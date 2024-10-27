package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jsmit257/userservice/shared/v1"
)

func (us *UserService) PatchContact(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	contact := &shared.Contact{}

	m := mtrcs.MustCurryWith(prometheus.Labels{"function": "PatchContact", "method": http.MethodPatch})

	userid := shared.UUID(chi.URLParam(r, "user_id"))
	if body, err := io.ReadAll(r.Body); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
	} else if err = json.Unmarshal(body, contact); err != nil {
		sc(http.StatusBadRequest).send(m, w, err)
		_, _ = w.Write([]byte(fmt.Sprintf("couldn't unmarshal: '%s'", body)))
	} else if err = us.Contacter.UpdateContact(r.Context(), userid, contact, cid()); err != nil {
		sc(http.StatusInternalServerError).send(m, w, err)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		sc(http.StatusOK).success(m, w)
	}
}
