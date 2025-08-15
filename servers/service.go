package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"vpn-manager/peers"
)

type IStore interface {
	GetAllActiveServers(ctx context.Context) ([]Server, error)
	GetByID(ctx context.Context, serverID string) (Server, error)
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

func (s *service) GetByID(ctx context.Context, serverID string) (Server, error) {
	return s.store.GetByID(ctx, serverID)
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

	fmt.Println(resp.StatusCode)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("delClient on server %s failed: status=%s", server.ID, resp.Status)
}
