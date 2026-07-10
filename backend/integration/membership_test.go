package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/incidrthreat/goshorten/backend/auth"
)

// TestSignupCreatesOwnerMembership verifies a new account owns its personal
// workspace via a membership row (Phase 15.1).
func TestSignupCreatesOwnerMembership(t *testing.T) {
	pgStore, authStore, _ := setup(t)
	defer pgStore.Pool.Close()

	u := makeUser(t, authStore, "owner-ms")
	ws := *u.DefaultWorkspaceID

	m, err := authStore.GetMembership(context.Background(), u.ID, ws)
	if err != nil {
		t.Fatalf("get membership: %v", err)
	}
	if m == nil {
		t.Fatal("expected an owner membership for the personal workspace")
	}
	if m.Role != auth.RoleOwner {
		t.Fatalf("personal workspace role = %q, want owner", m.Role)
	}
}

// TestLastOwnerInvariant verifies the last owner cannot be demoted or removed
// (Phase 15.4).
func TestLastOwnerInvariant(t *testing.T) {
	pgStore, authStore, _ := setup(t)
	defer pgStore.Pool.Close()

	owner := makeUser(t, authStore, "last-owner")
	ws := *owner.DefaultWorkspaceID
	ctx := context.Background()

	if err := authStore.UpdateMemberRole(ctx, ws, owner.ID, auth.RoleAdmin); !errors.Is(err, auth.ErrLastOwner) {
		t.Fatalf("demoting last owner: got %v, want ErrLastOwner", err)
	}
	if err := authStore.RemoveMember(ctx, ws, owner.ID); !errors.Is(err, auth.ErrLastOwner) {
		t.Fatalf("removing last owner: got %v, want ErrLastOwner", err)
	}

	// With a second owner present, demoting the first is allowed.
	other := makeUser(t, authStore, "co-owner")
	if err := authStore.UpsertMembership(ctx, other.ID, ws, auth.RoleOwner); err != nil {
		t.Fatalf("add co-owner: %v", err)
	}
	if err := authStore.UpdateMemberRole(ctx, ws, owner.ID, auth.RoleAdmin); err != nil {
		t.Fatalf("demote with two owners: %v", err)
	}
}

// TestInvitationAcceptGrantsScopedAccess verifies accepting an invite creates a
// membership that grants access to exactly that workspace and no other (15.7/15.10).
func TestInvitationAcceptGrantsScopedAccess(t *testing.T) {
	pgStore, authStore, _ := setup(t)
	defer pgStore.Pool.Close()

	inviter := makeUser(t, authStore, "inviter")
	invitee := makeUser(t, authStore, "invitee")
	other := makeUser(t, authStore, "outsider")
	wsInviter := *inviter.DefaultWorkspaceID
	wsOther := *other.DefaultWorkspaceID
	ctx := context.Background()

	// Before accepting, the invitee cannot access the inviter's workspace.
	if ok, _ := authStore.UserCanAccessWorkspace(ctx, invitee.ID, wsInviter); ok {
		t.Fatal("invitee should not have access before accepting")
	}

	inv, err := authStore.CreateInvitation(ctx, wsInviter, invitee.Email, auth.RoleMember, inviter.ID)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if _, err := authStore.AcceptInvitation(ctx, inv.Token, invitee.ID, invitee.Email); err != nil {
		t.Fatalf("accept invitation: %v", err)
	}

	// Now the invitee is a member of the inviter's workspace...
	if ok, _ := authStore.UserCanAccessWorkspace(ctx, invitee.ID, wsInviter); !ok {
		t.Fatal("invitee should have access after accepting")
	}
	role, _ := authStore.WorkspaceRole(ctx, invitee.ID, wsInviter)
	if role != auth.RoleMember {
		t.Fatalf("invitee role = %q, want member", role)
	}
	// ...but gains no access to an unrelated workspace.
	if ok, _ := authStore.UserCanAccessWorkspace(ctx, invitee.ID, wsOther); ok {
		t.Fatal("accepting an invite must not grant access to other workspaces")
	}

	// A mismatched email cannot accept.
	inv2, _ := authStore.CreateInvitation(ctx, wsInviter, "someone-else@example.test", auth.RoleViewer, inviter.ID)
	if _, err := authStore.AcceptInvitation(ctx, inv2.Token, other.ID, other.Email); err == nil {
		t.Fatal("accept with mismatched email unexpectedly succeeded")
	}
}

// TestTransferOwnership verifies ownership moves to a member and the previous
// owner is demoted (Phase 15.5).
func TestTransferOwnership(t *testing.T) {
	pgStore, authStore, _ := setup(t)
	defer pgStore.Pool.Close()

	owner := makeUser(t, authStore, "xfer-owner")
	member := makeUser(t, authStore, "xfer-member")
	ws := *owner.DefaultWorkspaceID
	ctx := context.Background()

	if err := authStore.UpsertMembership(ctx, member.ID, ws, auth.RoleMember); err != nil {
		t.Fatalf("add member: %v", err)
	}
	if err := authStore.TransferOwnership(ctx, ws, owner.ID, member.ID); err != nil {
		t.Fatalf("transfer ownership: %v", err)
	}

	if r, _ := authStore.WorkspaceRole(ctx, member.ID, ws); r != auth.RoleOwner {
		t.Fatalf("new owner role = %q, want owner", r)
	}
	if r, _ := authStore.WorkspaceRole(ctx, owner.ID, ws); r != auth.RoleAdmin {
		t.Fatalf("previous owner role = %q, want admin", r)
	}
	got, err := authStore.GetWorkspace(ctx, ws)
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got.OwnerID != member.ID {
		t.Fatalf("workspaces.owner_id = %d, want %d", got.OwnerID, member.ID)
	}
}
