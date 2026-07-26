package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type service struct {
	store               IStore
	peersService        IPeersService
	ServerPanelPassword string
	apiUrl              string
}

func NewService(store IStore, peersService IPeersService, ServerPanelPassword string, apiUrl string) *service {
	return &service{
		store:               store,
		peersService:        peersService,
		ServerPanelPassword: ServerPanelPassword,
		apiUrl:              apiUrl,
	}
}

type RegisterNewPeerOutput struct {
	Location string
	Uri      string
}

func (s *service) RegisterNewPeers(ctx context.Context, userID int64) error {
	client := http.Client{}

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
		loginResp, err := client.PostForm(server.ApiUrl+"/login",
			map[string][]string{
				"username": {server.Username},
				"password": {s.ServerPanelPassword},
			})
		if err != nil {
			log.Printf("error logging in to server: %s err: %v", server.ID, err)
			continue
		}

		defer loginResp.Body.Close()
		if loginResp.StatusCode != 200 {
			log.Printf("error logging in to server: %s bad gateway", server.ID)
			continue
		}

		setting, _ := json.Marshal(map[string]any{
			"clients": []map[string]any{{
				"id":         peer.UUID,
				"email":      peer.Email,
				"flow":       "",
				"enable":     true,
				"limitIp":    0,
				"totalGB":    0,
				"expiryTime": 0,
			}},
		})

		payload, _ := json.Marshal(map[string]any{
			"id":       server.InBoundID,
			"settings": string(setting),
		})

		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/panel/api/inbounds/addClient", server.ApiUrl), bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", loginResp.Header.Get("Set-Cookie"))

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error registering new peer on server: %s err: %v", server.ID, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("error registering new peer on server: %s bad status: %s", server.ID, resp.Status)
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

	return s.peersService.UpdateSubs(ctx, peer.ID, subs)
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
	case strings.TrimSpace(input.Username) == "":
		return fmt.Errorf("%w: username is required", ErrInvalidInput)
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
		Username:   strings.TrimSpace(input.Username),
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
	if input.Username != nil {
		fields["username"] = strings.TrimSpace(*input.Username)
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

const healthCheckTimeout = 7 * time.Second

// CheckHealth логинится в панель сервера и сообщает, отвечает ли она.
func (s *service) CheckHealth(ctx context.Context, serverID string) (Health, error) {
	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return Health{}, err
	}

	return s.checkServer(ctx, server), nil
}

// CheckAllHealth параллельно проверяет все серверы.
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

	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	form := url.Values{
		"username": {server.Username},
		"password": {s.ServerPanelPassword},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		server.ApiUrl+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		health.Error = "invalid api url"
		return health
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	started := time.Now()

	resp, err := (&http.Client{Timeout: healthCheckTimeout}).Do(req)
	health.LatencyMs = time.Since(started).Milliseconds()
	if err != nil {
		health.Error = "panel is unreachable"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		health.Error = fmt.Sprintf("panel responded with status %d", resp.StatusCode)
		return health
	}

	health.Reachable = true

	return health
}

func (s *service) DeletePeerFromServer(ctx context.Context, serverID, UUID string) error {
	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	client := http.Client{}

	loginResp, err := client.PostForm(server.ApiUrl+"/login",
		map[string][]string{
			"username": {server.Username},
			"password": {s.ServerPanelPassword},
		})
	if err != nil {
		return fmt.Errorf("login to server %s: %w", server.ID, err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		return fmt.Errorf("login to server %s failed: status=%s", server.ID, loginResp.Status)
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/panel/api/inbounds/%d/delClient/%s", server.ApiUrl, server.InBoundID, UUID), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", loginResp.Header.Get("Set-Cookie"))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delClient on server %s: %w", server.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("delClient on server %s failed: status=%s", server.ID, resp.Status)
}
