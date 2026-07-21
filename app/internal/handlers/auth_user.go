package handlers

import "net/http"

// @Summary      Current user
// @Description  Returns the authenticated user's ID and name. JWT only.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{id=int,name=string}
// @Failure      401
// @Router       /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := authenticatedUser(w, r)
	if !ok {
		return
	}

	writeJSON(w, struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   user.ID,
		Name: user.Name,
	})
}
