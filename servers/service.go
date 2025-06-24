package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"vpn-manager/peers"

	"github.com/google/uuid"
)

type IStore interface {
	GetAllActiveServers(ctx context.Context) ([]Server, error)
	GetByID(ctx context.Context, serverID string) (Server, error)
}

type IPeersService interface {
	CreatePeer(ctx context.Context, peer peers.CreatePeerInput) error
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

func (s *service) RegisterNewPeers(ctx context.Context, userID int64) ([]string, error) {
	client := http.Client{}

	servers, err := s.store.GetAllActiveServers(ctx)
	if err != nil {
		return []string{}, err
	}

	output := make([]string, 0, len(servers))

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

		uuid := uuid.New().String()
		email := uuid[:7]

		setting, _ := json.Marshal(map[string]any{
			"clients": []map[string]any{{
				"id":         uuid,
				"email":      email,
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
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("error registering new peer on server: %s bad status: %s", server.ID, resp.Status)
			continue
		}

		uri := fmt.Sprintf("vless://%s@%s:%d?encryption=none#%s", uuid, server.Ip, server.Port, server.Location)

		if err := s.peersService.CreatePeer(ctx, peers.CreatePeerInput{
			UserId:        userID,
			ServerId:      server.ID,
			Location:      server.Location,
			ConnectionURI: uri,
		}); err != nil {
			log.Printf("error creating peer for user %d on server %s: %v", userID, server.ID, err)
			continue
		}

		output = append(output, uri)
	}

	return output, nil
}

func (s *service) GetAllActiveServers(ctx context.Context) ([]Server, error) {
	return s.store.GetAllActiveServers(ctx)
}

func (s *service) GetByID(ctx context.Context, serverID string) (Server, error) {
	return s.store.GetByID(ctx, serverID)
}

func (s *service) DeletePeerFromServer(ctx context.Context, serverID, publicKey string) error {
	// Need implement
	return nil
}
