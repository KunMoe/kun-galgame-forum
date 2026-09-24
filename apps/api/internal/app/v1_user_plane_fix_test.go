package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
)

const (
	g6WorkLive    = 947000001
	g6WorkNSFW    = 947000002
	g6WorkHidden  = 947000003
	g6WorkOwner   = 947000004
	g6WorkExtra   = 947000005
	g6WorkUnknown = 947000099
	g6CoverA      = 11
	g6CoverB      = 12
	g6CoverOther  = 99

	g6FolderAliceDef  = 947200001
	g6FolderAlicePub  = 947200002
	g6FolderBobPriv   = 947200003
	g6FolderBobPub    = 947200004
	g6FolderBannedPub = 947200005
	g6FolderAlicePriv = 947200006

	g6AliasAlice = 947100001
	g6AliasBob   = 947100002
	g6AliasGone  = 947100099

	g6TokAlice   = "tok-alice"
	g6TokBob     = "tok-bob"
	g6TokStaff   = "tok-staff"
	g6TokOther   = "tok-other"
	g6TokNoscope = "tok-noscope"
	g6Tok429     = "tok-429"
	g6Tok429Q    = "tok-429-quota"
)

type g6Fix struct {
	*writeFix
	cat  *fakeCatalog
	user *g6User
}

type g6User struct {
	mu          sync.Mutex
	folders     map[int64]catalogclient.Folder
	items       map[int64][]catalogclient.FolderItem
	votes       map[int64]int64
	voteCount   map[int64]int
	play        map[string]map[int64]catalogclient.PlaytimeSelf
	states      map[string]map[int64]catalogclient.WorkStateRecord
	nextFolder  int64
	creates     int
	deletePlay  []int64
	otherApp    map[int64]int
	putStates   []int64
	modPatch    int
	modDelete   int
	putItems    []string
	deleteItems []string
	tallyErr    error
	itemsErr    error
	pubReads    int
	myReads     int
	playSweeps  int
	folderLists int
	holdReads   int
	previewRead int
	ownPatch403 bool
}

func newG6Fix(t *testing.T) *g6Fix {
	return newG6FixWith(t, nil)
}

func newG6FixDeny(t *testing.T) *g6Fix {
	return newG6FixWith(t, denyChecker{})
}

func newG6FixWith(t *testing.T, checker gate.Checker) *g6Fix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	g6bindToken(t, f, "sess-alice", w3UserAlice, g6TokAlice)
	g6bindToken(t, f, "sess-bob", w3UserBob, g6TokBob)
	g6bindToken(t, f, "sess-staff", w3UserStaff, g6TokStaff, "user", "moderator")
	g6bindToken(t, f, "sess-other", w3UserOther, g6TokOther)
	g6bindToken(t, f, "sess-noscope", w3UserAlice, g6TokNoscope)
	g6bindToken(t, f, "sess-429", w3UserAlice, g6Tok429)
	g6bindToken(t, f, "sess-429q", w3UserAlice, g6Tok429Q)
	g6bindToken(t, f, "sess-banned", w3UserBanned, g6TokAlice)

	cat := &fakeCatalog{
		rows:       map[int]client.CatalogWorkListItem{},
		details:    map[int]*client.CatalogWorkDetail{},
		movedWorks: map[int]int64{},
		works: []geWork{
			{id: g6WorkLive, name: "LiveG6", limit: "sfw", rating: "all_ages"},
			{id: g6WorkNSFW, name: "NsfwG6", limit: "nsfw", rating: "r18"},
			{id: g6WorkOwner, name: "OwnerG6", limit: "sfw", rating: "all_ages"},
			{id: g6WorkExtra, name: "ExtraG6", limit: "sfw", rating: "all_ages"},
		},
	}
	for _, w := range cat.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		cat.rows[w.id] = row
	}
	var hidden client.CatalogWorkListItem
	decodeInto(t, resourceHiddenJSON(g6WorkHidden, "HiddenG6"), &hidden)
	cat.rows[g6WorkHidden] = hidden
	cat.works = append(cat.works, geWork{id: g6WorkHidden, name: "HiddenG6", limit: "sfw", rating: "all_ages"})
	cat.details[g6WorkLive] = g6LiveDetail(t)

	user := newG6User()
	repo := galgameRepo.NewGalgameCollectionRepository(f.db)
	f.GalgameV1 = galgameapiv1.New(cat, nil, f.UserClient, f.rdb, geCDN).
		WithWork(f.db, user, nil, f.recordAward).
		WithUserPlane(f.TrustCheck, nil, repo)
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	gf := &g6Fix{writeFix: f, cat: cat, user: user}
	gf.cleanupG6(t)
	t.Cleanup(func() { gf.cleanupG6(t) })
	gf.seedG6(t)
	return gf
}

func g6bindToken(t *testing.T, f *writeFix, cookie string, userID int, token string, roles ...string) {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "n", Roles: roles},
		OAuthAccessToken: token,
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(cookie), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func g6LiveDetail(t *testing.T) *client.CatalogWorkDetail {
	t.Helper()
	raw := fmt.Sprintf(`{
		"id": %d, "display_name": "LiveG6", "latin": "liveg6", "olang": "ja",
		"content_rating": "all_ages", "created": "2026-01-01T00:00:00Z", "updated": "2026-01-02T00:00:00Z",
		"content_limit": "sfw", "claim": {"site": "kungal", "site_work_id": %d, "state": "live", "content_limit": "sfw"},
		"covers": [
			{"id": %d, "url": %q, "kind": "main", "source": "vndb", "width": 600, "height": 850, "thumbhash": "AbC+"},
			{"id": %d, "url": %q, "kind": "dig", "source": "vndb", "width": 800, "height": 450, "sexual": 0, "thumbhash": "sigO"}
		]
	}`, g6WorkLive, g6WorkLive, g6CoverA,
		fmt.Sprintf("https://image.other.example/aa/aa/%s.webp", g4PortraitHash),
		g6CoverB, fmt.Sprintf("https://image.other.example/bb/bb/%s.webp", g4SafeHash))
	var d client.CatalogWorkDetail
	decodeInto(t, raw, &d)
	return &d
}

func (f *g6Fix) cleanupG6(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM message WHERE link LIKE '/galgame/947000%' OR sender_id = 930000001 AND type = 'favorite'`,
		`DELETE FROM galgame_collection WHERE id BETWEEN 947100001 AND 947100099`,
		`DELETE FROM galgame WHERE id BETWEEN 947000001 AND 947000099`,
		`DELETE FROM galgame_view_daily WHERE entity_id BETWEEN 947000001 AND 947000099`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *g6Fix) seedG6(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("g6 seed: %v\n%s", err, q)
		}
	}
	run(`INSERT INTO galgame (id, view, like_count, favorite_count, created, updated, published, content_limit, creator_user_id, resource_update_time, resource_publish_banned)
		VALUES (?, 0, 0, 0, ?, ?, true, 'sfw', ?, ?, false),
		       (?, 0, 0, 2, ?, ?, true, 'sfw', ?, ?, false),
		       (?, 0, 0, 0, ?, ?, true, 'nsfw', NULL, ?, false)`,
		g6WorkLive, base, base, w3UserBob, base,
		g6WorkOwner, base, base, w3UserAlice, base,
		g6WorkNSFW, base, base, base)
	run(`INSERT INTO galgame_collection (id, user_id, name, description, visibility, is_default, item_count, catalog_folder_id, created, updated)
		VALUES (?, ?, '', '', 'public', true, 0, ?, ?, ?),
		       (?, ?, '', '', 'private', false, 0, ?, ?, ?)`,
		g6AliasAlice, w3UserAlice, int64(g6FolderAliceDef), base, base,
		g6AliasBob, w3UserBob, int64(g6FolderBobPriv), base, base)
}

func (f *g6Fix) call(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, idem, payload)
}

func g6Path(id int) string { return "/api/v1/works/" + idStr(id) }

func newG6User() *g6User {
	now := "2026-09-01T12:00:00Z"
	u := &g6User{
		folders:    map[int64]catalogclient.Folder{},
		items:      map[int64][]catalogclient.FolderItem{},
		votes:      map[int64]int64{},
		voteCount:  map[int64]int{g6CoverA: 3, g6CoverB: 1},
		play:       map[string]map[int64]catalogclient.PlaytimeSelf{},
		states:     map[string]map[int64]catalogclient.WorkStateRecord{},
		nextFolder: 947200100,
	}
	u.folders[g6FolderAliceDef] = catalogclient.Folder{
		ID: g6FolderAliceDef, OwnerUID: w3UserAlice, Name: "Alice default", Visibility: "public",
		IsDefault: true, ItemCount: 1, CreatedAt: now, UpdatedAt: "2026-09-02T12:00:00Z",
	}
	u.folders[g6FolderAlicePub] = catalogclient.Folder{
		ID: g6FolderAlicePub, OwnerUID: w3UserAlice, Name: "Alice other", Visibility: "public",
		ItemCount: 0, CreatedAt: now, UpdatedAt: "2026-09-03T12:00:00Z",
	}
	u.folders[g6FolderAlicePriv] = catalogclient.Folder{
		ID: g6FolderAlicePriv, OwnerUID: w3UserAlice, Name: "Alice private", Visibility: "private",
		ItemCount: 0, CreatedAt: now, UpdatedAt: "2026-09-01T12:00:00Z",
	}
	u.folders[g6FolderBobPriv] = catalogclient.Folder{
		ID: g6FolderBobPriv, OwnerUID: w3UserBob, Name: "Bob private", Visibility: "private",
		ItemCount: 0, CreatedAt: now, UpdatedAt: now,
	}
	u.folders[g6FolderBobPub] = catalogclient.Folder{
		ID: g6FolderBobPub, OwnerUID: w3UserBob, Name: "Bob public", Visibility: "public",
		ItemCount: 0, CreatedAt: now, UpdatedAt: now,
	}
	u.folders[g6FolderBannedPub] = catalogclient.Folder{
		ID: g6FolderBannedPub, OwnerUID: w3UserBanned, Name: "Banned public", Visibility: "public",
		ItemCount: 0, CreatedAt: now, UpdatedAt: now,
	}
	u.items[g6FolderAliceDef] = []catalogclient.FolderItem{{
		FolderID: g6FolderAliceDef, WorkID: g6WorkLive, CreatedAt: "2026-09-02T00:00:00Z", UpdatedAt: now,
	}}
	u.play[g6TokAlice] = map[int64]catalogclient.PlaytimeSelf{
		g6WorkLive:  {WorkID: g6WorkLive, Minutes: 120},
		g6WorkExtra: {WorkID: g6WorkExtra, Minutes: 5},
		g6WorkNSFW:  {WorkID: g6WorkNSFW, Minutes: 40},
	}
	done := "done"
	u.states[g6TokAlice] = map[int64]catalogclient.WorkStateRecord{
		g6WorkLive:  {WorkID: g6WorkLive, State: "doing"},
		g6WorkOwner: {WorkID: g6WorkOwner, State: done},
	}
	return u
}

func (u *g6User) uid(token string) int64 {
	switch token {
	case g6TokAlice, g6TokNoscope, g6Tok429, g6Tok429Q:
		return w3UserAlice
	case g6TokBob:
		return w3UserBob
	case g6TokStaff, "staff-token":
		return w3UserStaff
	case g6TokOther:
		return w3UserOther
	case "bob-token":
		return w3UserBob
	default:
		return w3UserAlice
	}
}

func (u *g6User) gate(token string) error {
	switch token {
	case g6TokNoscope:
		return catalogclient.ErrInsufficientScope
	case g6Tok429:
		return &catalogclient.UserAPIError{Status: http.StatusTooManyRequests, Message: "slow down", RetryAfter: "30"}
	case g6Tok429Q:
		return &catalogclient.UserAPIError{Status: http.StatusTooManyRequests, Message: "Daily quota exceeded", RetryAfter: "3600"}
	}
	return nil
}

func (u *g6User) MyFoldersContaining(_ context.Context, token string, workID int64) ([]catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	uid := u.uid(token)
	var out []catalogclient.Folder
	for fid, items := range u.items {
		f, ok := u.folders[fid]
		if !ok || f.OwnerUID != uid {
			continue
		}
		for _, it := range items {
			if it.WorkID == workID {
				out = append(out, f)
				break
			}
		}
	}
	return out, nil
}

func (u *g6User) MyFolderHoldings(_ context.Context, token string, workIDs []int64) ([]catalogclient.FolderHolding, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.holdReads++
	uid := u.uid(token)
	var out []catalogclient.FolderHolding
	for _, workID := range workIDs {
		h := catalogclient.FolderHolding{WorkID: workID}
		for fid, items := range u.items {
			if f, ok := u.folders[fid]; !ok || f.OwnerUID != uid {
				continue
			}
			for _, it := range items {
				if it.WorkID == workID {
					h.FolderIDs = append(h.FolderIDs, fid)
					break
				}
			}
		}
		if len(h.FolderIDs) > 0 {
			out = append(out, h)
		}
	}
	return out, nil
}
func (u *g6User) WorkCoversUser(context.Context, string, int64) ([]catalogclient.CoverTally, error) {
	return u.WorkCoverVotes(context.Background(), 0)
}
func (u *g6User) WorkCoverVotes(_ context.Context, workID int64) ([]catalogclient.CoverTally, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.tallyErr != nil {
		return nil, u.tallyErr
	}
	voted := u.votes[g6WorkLive]
	return []catalogclient.CoverTally{
		{ID: g6CoverA, ImageHash: g4PortraitHash, VoteCount: u.voteCount[g6CoverA], Voted: voted == g6CoverA},
		{ID: g6CoverB, ImageHash: g4SafeHash, VoteCount: u.voteCount[g6CoverB], Voted: voted == g6CoverB},
	}, nil
}
func (u *g6User) MyPlaytime(_ context.Context, token string, workID int64) (*catalogclient.PlaytimeSelf, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if other, ok := u.otherApp[workID]; ok {
		cp := catalogclient.PlaytimeSelf{Minutes: other}
		if row, ok := u.play[token][workID]; ok && row.Minutes > other {
			cp = row
		}
		return &cp, nil
	}
	if row, ok := u.play[token][workID]; ok {
		cp := row
		return &cp, nil
	}
	return nil, nil
}
func (u *g6User) MyWorkState(_ context.Context, token string, workID int64) (*catalogclient.WorkStateRecord, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if row, ok := u.states[token][workID]; ok {
		cp := row
		return &cp, nil
	}
	return nil, nil
}

func (u *g6User) VoteCover(_ context.Context, token string, workID, coverID int64) (*catalogclient.CoverVoteResult, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	prev := u.votes[workID]
	if prev != coverID {
		if prev != 0 {
			u.voteCount[prev] = max(u.voteCount[prev]-1, 0)
		}
		u.votes[workID] = coverID
		u.voteCount[coverID]++
	}
	return &catalogclient.CoverVoteResult{CoverID: coverID, VoteCount: int64(u.voteCount[coverID]), Voted: true}, nil
}
func (u *g6User) UnvoteCover(_ context.Context, token string, workID, coverID int64) (*catalogclient.CoverVoteResult, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.votes[workID] == coverID {
		delete(u.votes, workID)
		u.voteCount[coverID] = max(u.voteCount[coverID]-1, 0)
	}
	return &catalogclient.CoverVoteResult{CoverID: coverID, VoteCount: int64(u.voteCount[coverID]), Voted: false}, nil
}

func (u *g6User) ReportPlaytime(_ context.Context, token string, workID int64, report catalogclient.PlaytimeReport) (*catalogclient.PlaytimeRecord, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.play[token] == nil {
		u.play[token] = map[int64]catalogclient.PlaytimeSelf{}
	}
	u.play[token][workID] = catalogclient.PlaytimeSelf{WorkID: workID, Minutes: report.Minutes}
	return &catalogclient.PlaytimeRecord{WorkID: workID, Minutes: report.Minutes}, nil
}
func (u *g6User) DeleteMyPlaytime(_ context.Context, token string, workID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deletePlay = append(u.deletePlay, workID)
	if u.play[token] != nil {
		delete(u.play[token], workID)
	}
	return nil
}
func (u *g6User) ListMyPlaytime(_ context.Context, token, _ string, _ int) ([]catalogclient.PlaytimeRecord, string, error) {
	if err := u.gate(token); err != nil {
		return nil, "", err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.playSweeps++
	var out []catalogclient.PlaytimeRecord
	for _, row := range u.play[token] {
		out = append(out, catalogclient.PlaytimeRecord{WorkID: row.WorkID, Minutes: row.Minutes})
	}
	return out, "", nil
}
func (u *g6User) PutWorkState(_ context.Context, token string, workID int64, state string, completion *string) (*catalogclient.WorkStateRecord, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.putStates = append(u.putStates, workID)
	if u.states[token] == nil {
		u.states[token] = map[int64]catalogclient.WorkStateRecord{}
	}
	rec := catalogclient.WorkStateRecord{WorkID: workID, State: state, Completion: completion}
	u.states[token][workID] = rec
	return &rec, nil
}
func (u *g6User) DeleteWorkState(_ context.Context, token string, workID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.states[token] != nil {
		delete(u.states[token], workID)
	}
	return nil
}
func (u *g6User) ListMyWorkStates(_ context.Context, token, _ string, _ int) ([]catalogclient.WorkStateRecord, string, error) {
	if err := u.gate(token); err != nil {
		return nil, "", err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	var out []catalogclient.WorkStateRecord
	for _, row := range u.states[token] {
		out = append(out, row)
	}
	return out, "", nil
}

func (u *g6User) MyFolders(_ context.Context, token string) ([]catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.folderLists++
	uid := u.uid(token)
	var out []catalogclient.Folder
	for _, f := range u.folders {
		if f.OwnerUID == uid {
			out = append(out, f)
		}
	}
	return out, nil
}
func (u *g6User) MyFolder(_ context.Context, token string, folderID int64) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.OwnerUID != u.uid(token) {
		return nil, catalogclient.ErrNotFound
	}
	cp := f
	return &cp, nil
}
func (u *g6User) FolderPreviewItems(_ context.Context, token string, folderID int64, n int) ([]catalogclient.FolderItem, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.previewRead++
	items := u.items[folderID]
	if len(items) > n {
		items = items[:n]
	}
	return items, nil
}
func (u *g6User) MyFolderItems(_ context.Context, token string, folderID int64) ([]catalogclient.FolderItem, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.itemsErr != nil {
		return nil, u.itemsErr
	}
	u.myReads++
	return append([]catalogclient.FolderItem(nil), u.items[folderID]...), nil
}
func (u *g6User) PublicFolders(_ context.Context, ownerUID int64) ([]catalogclient.Folder, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	var out []catalogclient.Folder
	for _, f := range u.folders {
		if f.OwnerUID == ownerUID && f.Visibility == "public" {
			out = append(out, f)
		}
	}
	return out, nil
}
func (u *g6User) PublicFolder(_ context.Context, folderID int64) (*catalogclient.Folder, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.Visibility != "public" {
		return nil, catalogclient.ErrNotFound
	}
	cp := f
	return &cp, nil
}
func (u *g6User) PublicFolderItems(_ context.Context, folderID int64) ([]catalogclient.FolderItem, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.Visibility != "public" {
		return nil, catalogclient.ErrNotFound
	}
	u.pubReads++
	return append([]catalogclient.FolderItem(nil), u.items[folderID]...), nil
}
func (u *g6User) CreateFolderKeyed(_ context.Context, token string, in catalogclient.FolderWrite, _ string) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.creates++
	id := u.nextFolder
	u.nextFolder++
	name, desc, vis := "", "", "private"
	if in.Name != nil {
		name = *in.Name
	}
	if in.Description != nil {
		desc = *in.Description
	}
	if in.Visibility != nil {
		vis = *in.Visibility
	}
	isDef := in.IsDefault != nil && *in.IsDefault
	if isDef {
		uid := u.uid(token)
		for id, f := range u.folders {
			if f.OwnerUID == uid {
				f.IsDefault = false
				u.folders[id] = f
			}
		}
	}
	now := "2026-09-20T00:00:00Z"
	f := catalogclient.Folder{
		ID: id, OwnerUID: u.uid(token), Name: name, Description: desc,
		Visibility: vis, IsDefault: isDef, CreatedAt: now, UpdatedAt: now,
	}
	u.folders[id] = f
	return &f, nil
}
func (u *g6User) PatchFolder(_ context.Context, token string, folderID int64, in catalogclient.FolderWrite) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ownPatch403 {
		return nil, &catalogclient.UserAPIError{Status: http.StatusForbidden, Message: "no"}
	}
	f, ok := u.folders[folderID]
	if !ok || f.OwnerUID != u.uid(token) {
		return nil, catalogclient.ErrNotFound
	}
	if in.IsDefault != nil && !*in.IsDefault {
		return nil, &catalogclient.UserAPIError{Status: 422, Message: "is_default can only be set to true; set it on another folder to move it."}
	}
	if in.Name != nil {
		f.Name = *in.Name
	}
	if in.Description != nil {
		f.Description = *in.Description
	}
	if in.Visibility != nil {
		f.Visibility = *in.Visibility
	}
	if in.IsDefault != nil && *in.IsDefault {
		for id, o := range u.folders {
			if o.OwnerUID == f.OwnerUID {
				o.IsDefault = false
				u.folders[id] = o
			}
		}
		f.IsDefault = true
	}
	u.folders[folderID] = f
	return &f, nil
}
func (u *g6User) DeleteFolder(_ context.Context, token string, folderID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.OwnerUID != u.uid(token) {
		return catalogclient.ErrNotFound
	}
	if f.IsDefault {
		return &catalogclient.UserAPIError{Status: 422, Message: "the default folder cannot be deleted; make another folder the default first."}
	}
	delete(u.folders, folderID)
	delete(u.items, folderID)
	return nil
}
func (u *g6User) PutFolderItem(_ context.Context, token string, folderID, workID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.putItems = append(u.putItems, fmt.Sprintf("%d/%d", folderID, workID))
	f, ok := u.folders[folderID]
	if !ok || f.OwnerUID != u.uid(token) {
		return catalogclient.ErrNotFound
	}
	for _, it := range u.items[folderID] {
		if it.WorkID == workID {
			return nil
		}
	}
	u.items[folderID] = append(u.items[folderID], catalogclient.FolderItem{
		FolderID: folderID, WorkID: workID, CreatedAt: "2026-09-20T00:00:00Z", UpdatedAt: "2026-09-20T00:00:00Z",
	})
	f.ItemCount++
	u.folders[folderID] = f
	return nil
}
func (u *g6User) DeleteFolderItem(_ context.Context, token string, folderID, workID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deleteItems = append(u.deleteItems, fmt.Sprintf("%d/%d", folderID, workID))
	kept := u.items[folderID][:0]
	for _, it := range u.items[folderID] {
		if it.WorkID != workID {
			kept = append(kept, it)
		}
	}
	u.items[folderID] = kept
	if f, ok := u.folders[folderID]; ok {
		f.ItemCount = len(kept)
		u.folders[folderID] = f
	}
	return nil
}
func (u *g6User) ModeratePatchFolder(_ context.Context, token string, folderID int64, in catalogclient.FolderWrite) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.modPatch++
	f, ok := u.folders[folderID]
	if !ok {
		return nil, catalogclient.ErrNotFound
	}
	if in.Name != nil {
		f.Name = *in.Name
	}
	if in.Description != nil {
		f.Description = *in.Description
	}
	if in.Visibility != nil {
		f.Visibility = *in.Visibility
	}
	u.folders[folderID] = f
	return &f, nil
}
func (u *g6User) ModerateDeleteFolder(_ context.Context, token string, folderID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.modDelete++
	delete(u.folders, folderID)
	delete(u.items, folderID)
	return nil
}

func (u *g6User) folderCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.folders)
}

func g6col(id int64) string {
	return "/api/v1/collections/" + idStr(int(id))
}

func hasPrivateLeak(body map[string]any) bool {
	raw, _ := json.Marshal(body)
	s := strings.ToLower(string(raw))
	return strings.Contains(s, `"private"`) || strings.Contains(s, `"visibility":"private"`)
}
