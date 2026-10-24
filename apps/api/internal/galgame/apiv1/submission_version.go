package apiv1

import (
	"strconv"
	"strings"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

// submissionVersion is the forum's validator for a claim, not catalog's ETag.
// Catalog's queue rows carry neither an ETag nor the last event, so the web
// fetched a fresh ETag right before every write and the check could never
// fail. A claim with an event is versioned by it, since every transition
// writes one; a queue row only by its state.
func submissionVersion(it catalogclient.UserClaimItem) string {
	if it.LastEventID > 0 {
		return `"e` + strconv.FormatInt(it.LastEventID, 10) + `"`
	}
	return `"s` + it.ClaimState + `"`
}

// claimPrecondition checks If-Match against the claim the write path just
// read and answers the If-Match to forward: catalog's ETag of that read, so a
// change between the read and the write is still refused upstream.
func claimPrecondition(ifMatch string, cur catalogclient.UserClaimItem, curETag string) (string, *problem.Problem) {
	// Cloudflare gzips the JSON and turns a strong ETag into W/"...", which
	// the pattern refused as INVALID_FORMAT when the web echoed it back.
	v := strings.TrimPrefix(ifMatch, "W/")
	if v != "" && v != "*" && v != submissionVersion(cur) && v != `"s`+cur.ClaimState+`"` {
		return "", problem.New(problem.CodePreconditionFailed, "If-Match did not match the current representation.")
	}
	return ifMatchOrAny(curETag), nil
}
