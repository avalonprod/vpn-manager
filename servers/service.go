package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"vpn-manager/peers"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type IStore interface {
	GetAll(ctx context.Context) ([]Server, error)
	GetByID(ctx context.Context, serverID string) (Server, error)
}

type IPeersService interface {
	CreatePeer(ctx context.Context, peer peers.CreatePeerInput) (string, error)
}

type service struct {
	store        IStore
	peersService IPeersService
	apiUrl       string
}

func NewService(store IStore, peersService IPeersService, apiUrl string) *service {
	return &service{
		store:        store,
		peersService: peersService,
		apiUrl:       apiUrl,
	}
}

type RegisterResponse struct {
	IP              string `json:"ip"`
	ServerPublicKey string `json:"server_public_key"`
	Endpoint        string `json:"endpoint"`
	AllowedIPs      string `json:"allowed_ips"`
	DNS             string `json:"dns"`
}

type RegisterNewPeerOutput struct {
	Location  string
	ConfigUrl string
}

func (s *service) RegisterNewPeer(ctx context.Context, userID int64, serverID string) (RegisterNewPeerOutput, error) {
	p, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return RegisterNewPeerOutput{}, err
	}

	privateKey := p.String()
	publicKey := p.PublicKey().String()

	body, _ := json.Marshal(map[string]string{
		"client_public_key": publicKey,
		"client_name":       fmt.Sprintf("user_%d", userID),
	})

	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return RegisterNewPeerOutput{}, err
	}

	if !server.IsActive {
		return RegisterNewPeerOutput{}, ErrServerInactive
	}

	if server.ServerApiUrl == "" {
		return RegisterNewPeerOutput{}, errors.New("no active server available")
	}

	var r RegisterResponse

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/register-peer", server.ServerApiUrl), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", server.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return RegisterNewPeerOutput{}, err
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return RegisterNewPeerOutput{}, fmt.Errorf("registration failed: %s", string(bodyBytes))
	}

	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&r)

	conf := buildClientConf(privateKey, r)

	peerID, err := s.peersService.CreatePeer(ctx, peers.CreatePeerInput{
		UserId:    userID,
		ServerId:  server.ID,
		Config:    conf,
		Location:  server.Location,
		PublicKey: publicKey,
	})

	if err != nil {
		return RegisterNewPeerOutput{}, fmt.Errorf("failed to create peer: %w", err)
	}

	return RegisterNewPeerOutput{
		Location:  server.Location,
		ConfigUrl: fmt.Sprintf("%s/peers/%s", s.apiUrl, peerID),
	}, nil
}

func (s *service) GetAllServers(ctx context.Context) ([]Server, error) {
	return s.store.GetAll(ctx)
}

func (s *service) GetByID(ctx context.Context, serverID string) (Server, error) {
	return s.store.GetByID(ctx, serverID)
}

func (s *service) DeletePeerFromServer(ctx context.Context, serverID, publicKey string) error {
	server, err := s.store.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/peer?key=%s", server.ServerApiUrl, url.QueryEscape(publicKey)), nil)
	req.Header.Set("X-Auth-Token", server.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
