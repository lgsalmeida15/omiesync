package empresas

import (
	"context"
	"errors"
	"testing"
	"time"

	"omie-sync-api/internal/apperror"
)

// --- mock repository ---

type mockRepo struct {
	empresa   *Empresa
	empresas  []*Empresa
	total     int64
	insertErr error
	updateErr error
	getErr    error
}

func (m *mockRepo) Insert(_ context.Context, grupoID, nome, cnpj, appKey, appSecret string) (*Empresa, error) {
	if m.insertErr != nil {
		return nil, m.insertErr
	}
	return &Empresa{ID: "new-id", GrupoID: grupoID, Nome: nome, CNPJ: cnpj, AppKey: appKey, AppSecret: appSecret, Status: "ativa", StatusSync: "ativo"}, nil
}
func (m *mockRepo) GetByID(_ context.Context, _ string) (*Empresa, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.empresa == nil {
		return nil, errors.New("não encontrado")
	}
	return m.empresa, nil
}
func (m *mockRepo) List(_ context.Context, _ string, _, _ int32) ([]*Empresa, error) {
	return m.empresas, nil
}
func (m *mockRepo) Count(_ context.Context, _ string) (int64, error) { return m.total, nil }
func (m *mockRepo) Update(_ context.Context, id, nome, cnpj, appKey, appSecret string) (*Empresa, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &Empresa{ID: id, Nome: nome, AppKey: appKey, AppSecret: appSecret, Status: "ativa"}, nil
}
func (m *mockRepo) MarkDeletando(_ context.Context, _ string) error         { return nil }
func (m *mockRepo) InsertDeletionQueue(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockRepo) Reativar(_ context.Context, _ string) error               { return nil }
func (m *mockRepo) CancelDeletionQueue(_ context.Context, _ string) error    { return nil }
func (m *mockRepo) ListPendingDeletions(_ context.Context) ([]PendingDeletion, error) {
	return nil, nil
}
func (m *mockRepo) MarkDeletionExecuted(_ context.Context, _ string) error { return nil }

// --- testes ---

func activeEmpresa() *Empresa {
	return &Empresa{
		ID: "emp-1", GrupoID: "grp-1", Nome: "Acme",
		AppKey: "key123", AppSecret: "supersecret",
		Status: "ativa", StatusSync: "ativo",
	}
}

func TestService_Create_Success(t *testing.T) {
	svc := NewService(&mockRepo{})

	resp, err := svc.Create(context.Background(), "grp-1", CreateRequest{
		Nome: "Acme", AppKey: "key", AppSecret: "secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.AppSecret == "secret" {
		t.Error("app_secret não deve aparecer sem máscara na response")
	}
	if resp.AppSecret != "secr****" {
		t.Errorf("app_secret mascarado incorretamente: %q", resp.AppSecret)
	}
}

func TestService_Create_MissingFields(t *testing.T) {
	svc := NewService(&mockRepo{})

	cases := []CreateRequest{
		{Nome: "", AppKey: "k", AppSecret: "s"},
		{Nome: "x", AppKey: "", AppSecret: "s"},
		{Nome: "x", AppKey: "k", AppSecret: ""},
	}
	for _, req := range cases {
		_, err := svc.Create(context.Background(), "grp-1", req)
		if err == nil {
			t.Errorf("esperava erro para %+v", req)
		}
		ae, ok := apperror.IsAppError(err)
		if !ok || ae.Code != 422 {
			t.Errorf("esperava 422, got %v", err)
		}
	}
}

func TestService_Delete_Success(t *testing.T) {
	svc := NewService(&mockRepo{empresa: activeEmpresa()})

	err := svc.Delete(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestService_Delete_AlreadyDeletando(t *testing.T) {
	e := activeEmpresa()
	e.Status = "deletando"
	svc := NewService(&mockRepo{empresa: e})

	err := svc.Delete(context.Background(), "emp-1")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava 409, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})

	err := svc.Delete(context.Background(), "x")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava 404, got %v", err)
	}
}

func TestService_Reativar_Success(t *testing.T) {
	e := activeEmpresa()
	e.Status = "deletando"
	svc := NewService(&mockRepo{empresa: e})

	err := svc.Reativar(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("Reativar: %v", err)
	}
}

func TestService_Reativar_JaAtiva(t *testing.T) {
	svc := NewService(&mockRepo{empresa: activeEmpresa()})

	err := svc.Reativar(context.Background(), "emp-1")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava 409, got %v", err)
	}
}

func TestService_AppSecretNeverExposed(t *testing.T) {
	svc := NewService(&mockRepo{empresa: activeEmpresa()})

	resp, err := svc.GetByID(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if resp.AppSecret == "supersecret" {
		t.Error("app_secret exposto sem máscara")
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct{ in, want string }{
		{"supersecret", "supe****"},
		{"ab", "****"},
		{"abcd", "abcd****"},
		{"", "****"},
	}
	for _, tc := range cases {
		got := maskSecret(tc.in)
		if got != tc.want {
			t.Errorf("maskSecret(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
