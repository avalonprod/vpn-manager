package admin

import (
	"errors"
	"net/http"
	"vpn-manager/servers"

	"github.com/gorilla/mux"
)

type securityPayload struct {
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
	SNI       string `json:"sni"`
}

func (s securityPayload) toDomain() servers.Security {
	return servers.Security{
		PublicKey: s.PublicKey,
		ShortID:   s.ShortID,
		SNI:       s.SNI,
	}
}

type serverResponse struct {
	ID         string          `json:"id"`
	Location   string          `json:"location"`
	Username   string          `json:"username"`
	Host       string          `json:"host"`
	Port       int             `json:"port"`
	IP         string          `json:"ip"`
	ApiUrl     string          `json:"api_url"`
	InBoundID  int             `json:"inbound_id"`
	MaxClients int             `json:"max_clients"`
	IsActive   bool            `json:"is_active"`
	Security   securityPayload `json:"security"`
}

func toServerResponse(server servers.Server) serverResponse {
	return serverResponse{
		ID:         server.ID,
		Location:   server.Location,
		Username:   server.Username,
		Host:       server.Host,
		Port:       server.Port,
		IP:         server.Ip,
		ApiUrl:     server.ApiUrl,
		InBoundID:  server.InBoundID,
		MaxClients: server.MaxClients,
		IsActive:   server.IsActive,
		Security: securityPayload{
			PublicKey: server.Security.PublicKey,
			ShortID:   server.Security.ShortID,
			SNI:       server.Security.SNI,
		},
	}
}

func (h *Handler) handleListServers(w http.ResponseWriter, r *http.Request) {
	list, err := h.serversService.GetAll(r.Context())
	if err != nil {
		h.logger.Errorf("admin: failed to list servers: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load servers")
		return
	}

	// Показываем, сколько активных подключений приходится на каждую локацию.
	loadByLocation := map[string]int64{}
	if counts, err := h.peersService.CountByLocation(r.Context()); err == nil {
		for _, c := range counts {
			loadByLocation[c.Location] = c.Count
		}
	} else {
		h.logger.Errorf("admin: failed to count peers by location: %v", err)
	}

	type item struct {
		serverResponse
		Peers int64 `json:"peers"`
	}

	items := make([]item, 0, len(list))
	for _, server := range list {
		items = append(items, item{
			serverResponse: toServerResponse(server),
			Peers:          loadByLocation[server.Location],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createServerRequest struct {
	Location   string          `json:"location"`
	Username   string          `json:"username"`
	Host       string          `json:"host"`
	Port       int             `json:"port"`
	IP         string          `json:"ip"`
	ApiUrl     string          `json:"api_url"`
	InBoundID  int             `json:"inbound_id"`
	MaxClients int             `json:"max_clients"`
	IsActive   bool            `json:"is_active"`
	Security   securityPayload `json:"security"`
}

func (h *Handler) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req createServerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	server, err := h.serversService.Create(r.Context(), servers.CreateInput{
		Location:   req.Location,
		Username:   req.Username,
		Host:       req.Host,
		Port:       req.Port,
		Ip:         req.IP,
		ApiUrl:     req.ApiUrl,
		InBoundID:  req.InBoundID,
		MaxClients: req.MaxClients,
		IsActive:   req.IsActive,
		Security:   req.Security.toDomain(),
	})
	if err != nil {
		if errors.Is(err, servers.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Errorf("admin: failed to create server: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "server.create", server.ID, server.Location)

	writeJSON(w, http.StatusCreated, toServerResponse(server))
}

type updateServerRequest struct {
	Location   *string          `json:"location"`
	Username   *string          `json:"username"`
	Host       *string          `json:"host"`
	Port       *int             `json:"port"`
	IP         *string          `json:"ip"`
	ApiUrl     *string          `json:"api_url"`
	InBoundID  *int             `json:"inbound_id"`
	MaxClients *int             `json:"max_clients"`
	IsActive   *bool            `json:"is_active"`
	Security   *securityPayload `json:"security"`
}

func (h *Handler) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	serverID := mux.Vars(r)["id"]

	var req updateServerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := servers.UpdateInput{
		Location:   req.Location,
		Username:   req.Username,
		Host:       req.Host,
		Port:       req.Port,
		Ip:         req.IP,
		ApiUrl:     req.ApiUrl,
		InBoundID:  req.InBoundID,
		MaxClients: req.MaxClients,
		IsActive:   req.IsActive,
	}

	if req.Security != nil {
		security := req.Security.toDomain()
		input.Security = &security
	}

	server, err := h.serversService.Update(r.Context(), serverID, input)
	if err != nil {
		switch {
		case errors.Is(err, servers.ErrServerNotFound):
			writeError(w, http.StatusNotFound, "server not found")
		case errors.Is(err, servers.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Errorf("admin: failed to update server %s: %v", serverID, err)
			writeError(w, http.StatusInternalServerError, "failed to update server")
		}
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "server.update", server.ID, server.Location)

	writeJSON(w, http.StatusOK, toServerResponse(server))
}

func (h *Handler) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	serverID := mux.Vars(r)["id"]

	if err := h.serversService.Delete(r.Context(), serverID); err != nil {
		if errors.Is(err, servers.ErrServerNotFound) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		h.logger.Errorf("admin: failed to delete server %s: %v", serverID, err)
		writeError(w, http.StatusInternalServerError, "failed to delete server")
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "server.delete", serverID, "")

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) handleServersHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.serversService.CheckAllHealth(r.Context())
	if err != nil {
		h.logger.Errorf("admin: failed to check servers health: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check servers")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": health})
}

func (h *Handler) handleServerHealth(w http.ResponseWriter, r *http.Request) {
	serverID := mux.Vars(r)["id"]

	health, err := h.serversService.CheckHealth(r.Context(), serverID)
	if err != nil {
		if errors.Is(err, servers.ErrServerNotFound) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		h.logger.Errorf("admin: failed to check server %s: %v", serverID, err)
		writeError(w, http.StatusInternalServerError, "failed to check server")
		return
	}

	writeJSON(w, http.StatusOK, health)
}
