package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"vpn-manager/peers"

	"go.mongodb.org/mongo-driver/bson"
)

type IStore interface {
	GetAllActiveServers(ctx context.Context) ([]Server, error)
	GetAll(ctx context.Context) ([]Server, error)
	GetByID(ctx context.Context, serverID string) (Server, error)
	Create(ctx context.Context, server Server) (string, error)
	Update(ctx context.Context, serverID string, fields bson.M) error
	Delete(ctx context.Context, serverID string) error
	Count(ctx context.Context) (total int64, active int64, err error)
}

type IPeersService interface {
	Create(ctx context.Context, userID int64) (peers.Peer, error)
	UpdateSubs(ctx context.Context, id string, subs []peers.Sub) error
}

type IUsersService interface {
	IsBlocked(ctx context.Context, userID int64) (bool, error)
}

type service struct {
	store        IStore
	peersService IPeersService
	usersService IUsersService
	apiUrl       string
}

func NewService(store IStore, peersService IPeersService, usersService IUsersService, apiUrl string) *service {
	return &service{
		store:        store,
		peersService: peersService,
		usersService: usersService,
		apiUrl:       apiUrl,
	}
}

func authorize(req *http.Request, server Server) {
	req.Header.Set("Authorization", "Bearer "+server.AuthToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

type panelResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type PanelRejection struct {
	ServerID string
	Msg      string
}

func (e *PanelRejection) Error() string {
	return fmt.Sprintf("server %s rejected the request: %s", e.ServerID, e.Msg)
}

func callPanel(client *http.Client, req *http.Request, server Server) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("server %s is unreachable: %w", server.ID, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: server %s", ErrPanelUnauthorized, server.ID)
	case http.StatusNotFound:
		return fmt.Errorf("%w: server %s, check api_url and panel version", ErrPanelEndpointMissing, server.ID)
	default:
		return fmt.Errorf("server %s responded with status %s", server.ID, resp.Status)
	}

	var parsed panelResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("server %s returned an unexpected response", server.ID)
	}

	if !parsed.Success {
		return &PanelRejection{ServerID: server.ID, Msg: parsed.Msg}
	}

	return nil
}

type RegisterNewPeerOutput struct {
	Location string
	Uri      string
}

func (s *service) RegisterNewPeers(ctx context.Context, userID int64) error {
	blocked, err := s.usersService.IsBlocked(ctx, userID)
	if err != nil {
		return fmt.Errorf("check block status of user %d: %w", userID, err)
	}

	if blocked {
		return fmt.Errorf("%w: user %d", ErrUserBlocked, userID)
	}

	client := http.Client{Timeout: panelRequestTimeout}

	peer, err := s.peersService.Create(ctx, userID)
	if err != nil {
		return err
	}

	servers, err := s.store.GetAllActiveServers(ctx)
	if err != nil {
		return err
	}

	subs := make([]peers.Sub, 0, len(servers))

	for _, server := range servers {
		if server.AuthToken == "" {
			log.Printf("skipping server %s: auth token is not set", server.ID)
			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"client": map[string]any{
				"id":         peer.UUID,
				"email":      peer.Email,
				"flow":       "",
				"enable":     true,
				"limitIp":    0,
				"totalGB":    0,
				"expiryTime": 0,
			},
			"inboundIds": []int{server.InBoundID},
		})

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			fmt.Sprintf("%s/panel/api/clients/add", server.ApiUrl), bytes.NewReader(payload))
		if err != nil {
			log.Printf("error building request for server %s: %v", server.ID, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		authorize(req, server)

		if err := callPanel(&client, req, server); err != nil {
			log.Printf("error registering new peer: %v", err)
			continue
		}

		sub := fmt.Sprintf("vless://%s@%s:%d?security=reality&encryption=none&sni=%s&pbk=%s&sid=%s#%s", peer.UUID, server.Ip, server.Port, url.QueryEscape(server.Security.SNI), server.Security.PublicKey, server.Security.ShortID, server.Location)

		subs = append(subs, peers.Sub{
			Location: server.Location,
			ServerID: server.ID,
			URL:      sub,
			Enabled:  true,
		})
	}

	if err := s.peersService.UpdateSubs(ctx, peer.ID, subs); err != nil {
		return err
	}

	if len(subs) == 0 {
		return fmt.Errorf("%w: user %d", ErrNoServersAvailable, userID)
	}

	return nil
}

func (s *service) GetAllActiveServers(ctx context.Context) ([]Server, error) {
	return s.store.GetAllActiveServers(ctx)
}

func (s *service) GetAll(ctx context.Context) ([]Server, error) {
	return s.store.GetAll(ctx)
}

func (s *service) GetByID(ctx context.Context, serverID string) (Server, error) {
	return s.store.GetByID(ctx, serverID)
}

func (s *service) Count(ctx context.Context) (total int64, active int64, err error) {
	return s.store.Count(ctx)
}

func validateCreateInput(input CreateInput) error {
	switch {
	case strings.TrimSpace(input.Location) == "":
		return fmt.Errorf("%w: location is required", ErrInvalidInput)
	case strings.TrimSpace(input.Ip) == "":
		return fmt.Errorf("%w: ip is required", ErrInvalidInput)
	case strings.TrimSpace(input.ApiUrl) == "":
		return fmt.Errorf("%w: api_url is required", ErrInvalidInput)
	case strings.TrimSpace(input.AuthToken) == "":
		return fmt.Errorf("%w: auth_token is required", ErrInvalidInput)
	case input.Port <= 0 || input.Port > 65535:
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	case input.MaxClients < 0:
		return fmt.Errorf("%w: max_clients must not be negative", ErrInvalidInput)
	}

	return nil
}

func (s *service) Create(ctx context.Context, input CreateInput) (Server, error) {
	if err := validateCreateInput(input); err != nil {
		return Server{}, err
	}

	server := Server{
		Location:   strings.TrimSpace(input.Location),
		AuthToken:  strings.TrimSpace(input.AuthToken),
		Host:       strings.TrimSpace(input.Host),
		Port:       input.Port,
		Ip:         strings.TrimSpace(input.Ip),
		ApiUrl:     strings.TrimRight(strings.TrimSpace(input.ApiUrl), "/"),
		InBoundID:  input.InBoundID,
		MaxClients: input.MaxClients,
		IsActive:   input.IsActive,
		Security:   input.Security,
	}

	id, err := s.store.Create(ctx, server)
	if err != nil {
		return Server{}, err
	}

	server.ID = id

	return server, nil
}

func (s *service) Update(ctx context.Context, serverID string, input UpdateInput) (Server, error) {
	fields := bson.M{}

	if input.Location != nil {
		location := strings.TrimSpace(*input.Location)
		if location == "" {
			return Server{}, fmt.Errorf("%w: location must not be empty", ErrInvalidInput)
		}
		fields["location"] = location
	}

	if input.AuthToken != nil {
		if token := strings.TrimSpace(*input.AuthToken); token != "" {
			fields["auth_token"] = token
		}
	}
	if input.Host != nil {
		fields["host"] = strings.TrimSpace(*input.Host)
	}
	if input.Port != nil {
		if *input.Port <= 0 || *input.Port > 65535 {
			return Server{}, fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
		}
		fields["port"] = *input.Port
	}
	if input.Ip != nil {
		fields["ip"] = strings.TrimSpace(*input.Ip)
	}
	if input.ApiUrl != nil {
		fields["api_url"] = strings.TrimRight(strings.TrimSpace(*input.ApiUrl), "/")
	}
	if input.InBoundID != nil {
		fields["inbound_id"] = *input.InBoundID
	}
	if input.MaxClients != nil {
		if *input.MaxClients < 0 {
			return Server{}, fmt.Errorf("%w: max_clients must not be negative", ErrInvalidInput)
		}
		fields["max_clients"] = *input.MaxClients
	}
	if input.IsActive != nil {
		fields["is_active"] = *input.IsActive
	}
	if input.Security != nil {
		fields["security"] = *input.Security
	}

	if err := s.store.Update(ctx, serverID, fields); err != nil {
		return Server{}, err
	}

	return s.store.GetByID(ctx, serverID)
}

func (s *service) Delete(ctx context.Context, serverID string) error {
	return s.store.Delete(ctx, serverID)
}

const (
	healthCheckTimeout = 7 * time.Second

	panelRequestTimeout = 15 * time.Second
)

func (s *service) CheckHealth(ctx context.Context, serverID string) (Health, error) {
	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return Health{}, err
	}

	return s.checkServer(ctx, server), nil
}

func (s *service) CheckAllHealth(ctx context.Context) ([]Health, error) {
	servers, err := s.store.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]Health, len(servers))

	var wg sync.WaitGroup
	for i, server := range servers {
		wg.Add(1)
		go func(i int, server Server) {
			defer wg.Done()
			results[i] = s.checkServer(ctx, server)
		}(i, server)
	}
	wg.Wait()

	return results, nil
}

func (s *service) checkServer(ctx context.Context, server Server) Health {
	health := Health{ServerID: server.ID, Location: server.Location}

	if server.AuthToken == "" {
		health.Error = "auth token is not set"
		return health
	}

	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/panel/api/inbounds/list", server.ApiUrl), nil)
	if err != nil {
		health.Error = "invalid api url"
		return health
	}
	authorize(req, server)

	started := time.Now()
	err = callPanel(&http.Client{Timeout: healthCheckTimeout}, req, server)
	health.LatencyMs = time.Since(started).Milliseconds()

	if err != nil {
		health.Error = err.Error()
		return health
	}

	health.Reachable = true

	return health
}

type RevocationResult struct {
	Revoked int
	Failed  []string
}

func (s *service) RevokeAccessEverywhere(ctx context.Context, email string) (RevocationResult, error) {
	result := RevocationResult{Failed: []string{}}

	serverList, err := s.store.GetAll(ctx)
	if err != nil {
		return result, err
	}

	for _, server := range serverList {
		if err := s.deleteClient(ctx, server, email); err != nil {
			log.Printf("revoke access for %s on server %s: %v", email, server.ID, err)
			result.Failed = append(result.Failed, server.Location)
			continue
		}
		result.Revoked++
	}

	return result, nil
}

func (s *service) DeletePeerFromServer(ctx context.Context, serverID, email string) error {
	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	return s.deleteClient(ctx, server, email)
}

func (s *service) deleteClient(ctx context.Context, server Server, email string) error {
	if server.AuthToken == "" {
		return fmt.Errorf("%w: server %s has no auth token", ErrInvalidInput, server.ID)
	}

	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("%w: client email is required", ErrInvalidInput)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/panel/api/clients/del/%s", server.ApiUrl, url.PathEscape(email)), nil)
	if err != nil {
		return fmt.Errorf("delete client on server %s: %w", server.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	authorize(req, server)

	err = callPanel(&http.Client{Timeout: panelRequestTimeout}, req, server)

	var rejection *PanelRejection
	if errors.As(err, &rejection) && clientAlreadyGone(rejection.Msg) {
		return nil
	}

	return err
}

func clientAlreadyGone(panelMsg string) bool {
	msg := strings.ToLower(panelMsg)

	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "not exist")
}
