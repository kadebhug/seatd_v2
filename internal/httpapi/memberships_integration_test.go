//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
)

func TestMemberManagementProvisionByEmail(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createDeviceAuthzFixture(t, ctx, pool)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)
	email := "invited-" + uuid.NewString() + "@example.test"

	createBody := `{"scope":"location","locationId":"` + fixture.OtherLocation.ID.String() + `","email":"` + email + `","displayName":"Invited Manager","role":"location_manager"}`
	createRec := assertDeviceRequestStatus(t, handler, http.MethodPost, "/v1/memberships", fixture, uuid.Nil, fixture.OwnerRef, createBody, http.StatusCreated)
	var created membershipDTO
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created membership: %v", err)
	}
	if created.ID == "" || created.LocationID == nil || *created.LocationID != fixture.OtherLocation.ID.String() {
		t.Fatalf("created membership = %#v, want other location membership", created)
	}
	if len(created.Permissions) == 0 {
		t.Fatalf("created permissions = %#v, want effective permissions", created.Permissions)
	}

	service := identity.NewService(pool)
	profile, err := service.ResolveExternalIdentity(ctx, identity.ResolveExternalIdentityParams{
		DisplayName:   "Invited Manager",
		Email:         email,
		EmailVerified: true,
		External: identity.ExternalIdentity{
			Issuer:  "https://issuer.example.test",
			Subject: "invited-" + uuid.NewString(),
		},
	})
	if err != nil {
		t.Fatalf("resolve invited external identity: %v", err)
	}
	session, err := service.CreateWebSession(ctx, identity.CreateWebSessionParams{
		UserProfileID: profile.User.ID,
		TTL:           time.Hour,
	})
	if err != nil {
		t.Fatalf("create invited web session: %v", err)
	}
	if len(session.Memberships) != 1 || session.Memberships[0].Role != identity.RoleLocationManager {
		t.Fatalf("session memberships = %#v, want invited location manager", session.Memberships)
	}

	updateBody := `{"role":"waiter"}`
	updateRec := assertDeviceRequestStatus(t, handler, http.MethodPut, "/v1/memberships/location/"+created.ID, fixture, uuid.Nil, fixture.OwnerRef, updateBody, http.StatusOK)
	var updated membershipDTO
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated membership: %v", err)
	}
	if updated.Role != identity.RoleWaiter {
		t.Fatalf("updated role = %q, want waiter", updated.Role)
	}

	disableRec := assertDeviceRequestStatus(t, handler, http.MethodPost, "/v1/memberships/location/"+created.ID+"/disable", fixture, uuid.Nil, fixture.OwnerRef, "", http.StatusOK)
	var disabled membershipDTO
	if err := json.Unmarshal(disableRec.Body.Bytes(), &disabled); err != nil {
		t.Fatalf("decode disabled membership: %v", err)
	}
	if disabled.DisabledAt == nil {
		t.Fatalf("disabledAt = nil, want timestamp")
	}

	listActiveRec := assertDeviceRequestStatus(t, handler, http.MethodGet, "/v1/memberships", fixture, uuid.Nil, fixture.OwnerRef, "", http.StatusOK)
	var active struct {
		Memberships []membershipDTO `json:"memberships"`
	}
	if err := json.Unmarshal(listActiveRec.Body.Bytes(), &active); err != nil {
		t.Fatalf("decode active memberships: %v", err)
	}
	for _, membership := range active.Memberships {
		if membership.ID == created.ID {
			t.Fatalf("active memberships include disabled membership %#v", membership)
		}
	}

	assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.OwnerRef, "membership.create", 1)
	assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.OwnerRef, "membership.update_role", 1)
	assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.OwnerRef, "membership.disable", 1)
}

func TestMemberManagementProtectsLastOrganisationOwner(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createDeviceAuthzFixture(t, ctx, pool)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)

	listRec := assertDeviceRequestStatus(t, handler, http.MethodGet, "/v1/memberships", fixture, uuid.Nil, fixture.OwnerRef, "", http.StatusOK)
	var list struct {
		Memberships []membershipDTO `json:"memberships"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode memberships: %v", err)
	}
	var ownerID string
	for _, membership := range list.Memberships {
		if membership.Scope == identity.MembershipScopeOrganisation && membership.Role == identity.RoleOrganisationOwner {
			ownerID = membership.ID
			break
		}
	}
	if ownerID == "" {
		t.Fatal("owner membership not found")
	}

	assertDeviceRequestStatus(t, handler, http.MethodPost, "/v1/memberships/organisation/"+ownerID+"/disable", fixture, uuid.Nil, fixture.OwnerRef, "", http.StatusBadRequest)
	assertDeviceRequestStatus(t, handler, http.MethodPut, "/v1/memberships/organisation/"+ownerID, fixture, uuid.Nil, fixture.OwnerRef, `{"role":"read_only"}`, http.StatusBadRequest)
}
