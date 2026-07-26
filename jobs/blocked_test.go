package jobs

import (
	"context"
	"errors"
	"testing"
	"vpn-manager/peers"
	"vpn-manager/servers"
	"vpn-manager/users"
)

type stubUsers struct {
	all    []users.User
	filter users.ListFilter
}

func (s *stubUsers) List(_ context.Context, f users.ListFilter) ([]users.User, int64, error) {
	s.filter = f

	if f.Offset >= len(s.all) {
		return nil, int64(len(s.all)), nil
	}

	end := min(f.Offset+f.Limit, len(s.all))

	return s.all[f.Offset:end], int64(len(s.all)), nil
}

type stubPeers struct {
	byUser      map[int64]peers.Peer
	deactivated []int64
}

func (s *stubPeers) GetPeerByUserID(_ context.Context, userID int64) (peers.Peer, error) {
	peer, ok := s.byUser[userID]
	if !ok {
		return peers.Peer{}, errors.New("not found")
	}
	return peer, nil
}

func (s *stubPeers) DeactivatePeer(_ context.Context, userID int64) error {
	s.deactivated = append(s.deactivated, userID)
	return nil
}

type stubServers struct {
	revoked []string
	failOn  string
}

func (s *stubServers) RevokeAccessEverywhere(_ context.Context, email string) (servers.RevocationResult, error) {
	s.revoked = append(s.revoked, email)

	if email == s.failOn {
		return servers.RevocationResult{Failed: []string{"Almaty"}}, nil
	}

	return servers.RevocationResult{Revoked: 1, Failed: []string{}}, nil
}

type nopLogger struct{}

func (nopLogger) Debug(...any)          {}
func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Info(...any)           {}
func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Warn(...any)           {}
func (nopLogger) Warnf(string, ...any)  {}
func (nopLogger) Error(...any)          {}
func (nopLogger) Errorf(string, ...any) {}

func TestReconcileRevokesEveryBlockedUser(t *testing.T) {
	u := &stubUsers{all: []users.User{{ID: 1}, {ID: 2}}}
	p := &stubPeers{byUser: map[int64]peers.Peer{
		1: {UserID: 1, Email: "aaa1111", IsActive: true},
		2: {UserID: 2, Email: "bbb2222", IsActive: true},
	}}
	s := &stubServers{}

	reconcileBlocked(context.Background(), u, p, s, nopLogger{})

	if len(s.revoked) != 2 {
		t.Errorf("revoked = %v, want both blocked users", s.revoked)
	}

	if len(p.deactivated) != 2 {
		t.Errorf("deactivated = %v, want both peers", p.deactivated)
	}
}

func TestReconcileQueriesOnlyBlockedUsers(t *testing.T) {
	u := &stubUsers{}

	reconcileBlocked(context.Background(), u, &stubPeers{}, &stubServers{}, nopLogger{})

	if u.filter.Blocked != users.BlockedOnly {
		t.Errorf("filter.Blocked = %q, want BlockedOnly", u.filter.Blocked)
	}
}

func TestReconcileKeepsGoingAfterPartialFailure(t *testing.T) {
	u := &stubUsers{all: []users.User{{ID: 1}, {ID: 2}}}
	p := &stubPeers{byUser: map[int64]peers.Peer{
		1: {UserID: 1, Email: "aaa1111"},
		2: {UserID: 2, Email: "bbb2222"},
	}}
	s := &stubServers{failOn: "aaa1111"}

	reconcileBlocked(context.Background(), u, p, s, nopLogger{})

	if len(s.revoked) != 2 {
		t.Errorf("revoked = %v, want the second user attempted despite the first failing", s.revoked)
	}
}

func TestReconcileSkipsUsersWithoutPeer(t *testing.T) {
	u := &stubUsers{all: []users.User{{ID: 1}}}
	s := &stubServers{}

	reconcileBlocked(context.Background(), u, &stubPeers{byUser: map[int64]peers.Peer{}}, s, nopLogger{})

	if len(s.revoked) != 0 {
		t.Errorf("revoked = %v, want nothing for a user without a peer", s.revoked)
	}
}
