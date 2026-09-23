package app

import (
	"net/http"
	"testing"
)

func TestV1GetCreatorStatus(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	resp, body := f.call(t, http.MethodGet, mePath+"/creator-status", "/me/creator-status", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if body["object"] != "creator_status" {
		t.Fatalf("%+v", body)
	}
	elig, _ := body["eligibility"].(map[string]any)
	if _, ok := elig["long_review_count"]; !ok {
		t.Fatalf("eligibility %+v", elig)
	}
}

func TestV1CreatorStatusMoemoepointFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.moeFail.Store(true)
	resp, body := f.call(t, http.MethodGet, mePath+"/creator-status", "/me/creator-status", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1CreatorUnknownStateIs500(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	f.creatorState.Store("weird")
	resp, body := f.call(t, http.MethodGet, mePath+"/creator-status", "/me/creator-status", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1CreateCreatorApplication(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(810), nil, map[string]any{"statement": "hi"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Location"); got != "/api/v1/me/creator-status" {
		t.Fatalf("Location %q", got)
	}
	if body["object"] != "creator_application" || body["state"] != "pending" {
		t.Fatalf("%+v", body)
	}
}

func TestV1CreateCreatorApplicationIneligible(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(0)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(811), nil, map[string]any{"statement": ""})
	mustCode(t, resp, body, http.StatusForbidden, "CREATOR_INELIGIBLE")
}

func TestV1CreateCreatorApplicationAlreadyCreator(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	f.creatorPost.Store(17001)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(812), nil, map[string]any{"statement": "x"})
	mustCode(t, resp, body, http.StatusConflict, "INVALID_STATE_TRANSITION")
}

func TestV1CreateCreatorApplicationAlreadyExists(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	f.creatorPost.Store(17002)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(813), nil, map[string]any{"statement": "x"})
	mustCode(t, resp, body, http.StatusConflict, "ALREADY_EXISTS")
}

func TestV1CreateCreatorApplicationCooldown(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	f.creatorPost.Store(17003)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(814), nil, map[string]any{"statement": "x"})
	mustCode(t, resp, body, http.StatusConflict, "CREATOR_APPLICATION_COOLDOWN")
}

func TestV1CreateCreatorApplicationOtherFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.moeBalance.Store(2000)
	f.creatorPost.Store(10)
	resp, body := f.call(t, http.MethodPost, mePath+"/creator-applications", "/me/creator-applications", "sess-alice", keyUUID(815), nil, map[string]any{"statement": "x"})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}
