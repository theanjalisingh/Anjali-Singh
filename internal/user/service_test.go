package user

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

type fakeStore struct {
	users         map[string]User
	roles         map[string]Role
	organizations map[string]bool
	emails        map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users: map[string]User{
			"user-1": {
				UserID:         "user-1",
				UserName:       "Org User",
				EmailID:        "org.user@example.com",
				OrganizationID: "org-1",
				IsActive:       true,
				IsApproved:     true,
				Roles:          []Role{{RoleID: "role-client", RoleCode: "CLIENT", RoleName: "Client"}},
			},
		},
		roles: map[string]Role{
			"role-client": {RoleID: "role-client", RoleCode: "CLIENT", RoleName: "Client"},
			"role-admin":  {RoleID: "role-admin", RoleCode: "ORG_ADMIN", RoleName: "Org Admin"},
			"role-super":  {RoleID: "role-super", RoleCode: "SUPER_ADMIN", RoleName: "Super Admin"},
		},
		organizations: map[string]bool{
			"org-1": true,
			"org-2": true,
		},
		emails: map[string]string{
			"org.user@example.com": "user-1",
		},
	}
}

func (f *fakeStore) GetByID(_ context.Context, userID string) (*User, error) {
	record, ok := f.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	copy := record
	return &copy, nil
}

func (f *fakeStore) List(_ context.Context) ([]User, error) {
	result := make([]User, 0, len(f.users))
	for _, record := range f.users {
		result = append(result, record)
	}
	return result, nil
}

func (f *fakeStore) Create(_ context.Context, input createUserInput) (*User, error) {
	email := ""
	if input.EmailID != nil {
		email = *input.EmailID
	}
	if _, exists := f.emails[email]; exists {
		return nil, ErrDuplicateEmail
	}

	created := User{
		UserID:         "user-new",
		UserName:       derefString(input.UserName),
		EmailID:        email,
		OrganizationID: derefString(input.OrganizationID),
		IsActive:       input.IsActive,
		IsApproved:     input.IsApproved,
		Roles:          []Role{f.roles[input.RoleID]},
	}
	f.users[created.UserID] = created
	f.emails[email] = created.UserID
	return &created, nil
}

func (f *fakeStore) Update(_ context.Context, input manageUserInput) (*User, error) {
	record, ok := f.users[derefString(input.UserID)]
	if !ok {
		return nil, ErrUserNotFound
	}
	if input.UserName != nil {
		record.UserName = *input.UserName
	}
	if input.EmailID != nil {
		record.EmailID = *input.EmailID
	}
	record.IsActive = input.IsActive
	f.users[record.UserID] = record
	return &record, nil
}

func (f *fakeStore) Delete(_ context.Context, userID, _ string) (*DeleteUserResponse, error) {
	record, ok := f.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	record.IsDeleted = true
	record.IsActive = false
	f.users[userID] = record
	return &DeleteUserResponse{
		UserID:    record.UserID,
		UserName:  record.UserName,
		EmailID:   record.EmailID,
		IsActive:  false,
		IsDeleted: true,
	}, nil
}

func (f *fakeStore) GetRoleByID(_ context.Context, roleID string) (*roleRecord, error) {
	role, ok := f.roles[roleID]
	if !ok {
		return nil, ErrInvalidRole
	}
	return &roleRecord{RoleID: role.RoleID, RoleCode: role.RoleCode, RoleName: role.RoleName}, nil
}

func (f *fakeStore) OrganizationExists(_ context.Context, organizationID string) (bool, error) {
	return f.organizations[organizationID], nil
}

func (f *fakeStore) EmailExists(_ context.Context, emailID, excludeUserID string) (bool, error) {
	userID, ok := f.emails[emailID]
	if !ok {
		return false, nil
	}
	return userID != excludeUserID, nil
}

func (f *fakeStore) SetUserRole(_ context.Context, userID, roleID, _ string) error {
	record, ok := f.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	record.Roles = []Role{f.roles[roleID]}
	f.users[userID] = record
	return nil
}

func testService(t *testing.T, store Store) *Service {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewService(store, logger)
}

func TestOrgAdminCannotAssignSuperAdmin(t *testing.T) {
	store := newFakeStore()
	svc := testService(t, store)
	active := true

	_, err := svc.Create(context.Background(), Actor{
		UserID:         "admin-1",
		OrganizationID: "org-1",
		Roles:          []string{"ORG_ADMIN"},
	}, CreateUserRequest{
		UserName:       "New User",
		EmailID:        "new.user@example.com",
		Password:       "Password123",
		OrganizationID: "org-1",
		RoleID:         "role-super",
		IsActive:       &active,
	})
	if err != ErrCannotAssignRole {
		t.Fatalf("Create() error = %v, want %v", err, ErrCannotAssignRole)
	}
}

func TestOrgAdminListFiltersByOrganization(t *testing.T) {
	store := newFakeStore()
	store.users["user-2"] = User{
		UserID:         "user-2",
		UserName:       "Other Org User",
		EmailID:        "other@example.com",
		OrganizationID: "org-2",
		IsActive:       true,
	}

	svc := testService(t, store)
	users, err := svc.List(context.Background(), Actor{
		UserID:         "admin-1",
		OrganizationID: "org-1",
		Roles:          []string{"ORG_ADMIN"},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 1 || users[0].UserID != "user-1" {
		t.Fatalf("List() = %#v, want only org-1 user", users)
	}
}

func TestClientCannotManageUsers(t *testing.T) {
	svc := testService(t, newFakeStore())
	_, err := svc.List(context.Background(), Actor{Roles: []string{"CLIENT"}})
	if err != ErrForbidden {
		t.Fatalf("List() error = %v, want %v", err, ErrForbidden)
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
