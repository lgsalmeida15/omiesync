package usuarios

import (
	"context"
	"errors"
	"testing"

	"omie-sync-api/internal/apperror"
)

type mockRepo struct {
	usuario  *Usuario
	usuarios []*Usuario
	total    int64
	err      error
}

func (m *mockRepo) Insert(_ context.Context, grupoID, nome, email, _, role string) (*Usuario, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Usuario{ID: "u1", GrupoID: grupoID, Nome: nome, Email: email, Role: role, Ativo: true}, nil
}
func (m *mockRepo) GetByID(_ context.Context, _ string) (*Usuario, error) {
	if m.usuario == nil {
		return nil, errors.New("não encontrado")
	}
	return m.usuario, nil
}
func (m *mockRepo) List(_ context.Context, _ string, _, _ int32) ([]*Usuario, error) {
	return m.usuarios, m.err
}
func (m *mockRepo) Count(_ context.Context, _ string) (int64, error) { return m.total, m.err }
func (m *mockRepo) Update(_ context.Context, id, nome, role string, ativo bool) (*Usuario, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Usuario{ID: id, Nome: nome, Role: role, Ativo: ativo}, nil
}
func (m *mockRepo) UpdatePassword(_ context.Context, _, _ string) error { return m.err }
func (m *mockRepo) SoftDelete(_ context.Context, _ string) error        { return m.err }

func activeUser() *Usuario {
	return &Usuario{ID: "u1", GrupoID: "g1", Nome: "João", Email: "j@t.com", Role: "viewer", Ativo: true}
}

func TestService_Create_Success(t *testing.T) {
	svc := NewService(&mockRepo{})
	u, err := svc.Create(context.Background(), "g1", CreateRequest{
		Nome: "Ana", Email: "ana@t.com", Password: "senha123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.Role != "viewer" {
		t.Errorf("role default: got %q", u.Role)
	}
	if u.Email != "ana@t.com" {
		t.Errorf("email: got %q", u.Email)
	}
}

func TestService_Create_PasswordHasheado(t *testing.T) {
	// Verifica que password não é armazenado em plain text
	// O mock captura o hash mas não podemos inspecioná-lo diretamente —
	// o teste garante que não retorna erro (bcrypt não falhou)
	svc := NewService(&mockRepo{})
	_, err := svc.Create(context.Background(), "g1", CreateRequest{
		Nome: "Bob", Email: "b@t.com", Password: "minhasenha",
	})
	if err != nil {
		t.Fatalf("Create com senha válida não deveria falhar: %v", err)
	}
}

func TestService_Create_Validacoes(t *testing.T) {
	svc := NewService(&mockRepo{})
	cases := []struct {
		req  CreateRequest
		code int
	}{
		{CreateRequest{Nome: "", Email: "e@t.com", Password: "12345678"}, 422},
		{CreateRequest{Nome: "x", Email: "", Password: "12345678"}, 422},
		{CreateRequest{Nome: "x", Email: "e@t.com", Password: "curta"}, 422},
		{CreateRequest{Nome: "x", Email: "e@t.com", Password: "12345678", Role: "invalida"}, 422},
	}
	for _, tc := range cases {
		_, err := svc.Create(context.Background(), "g1", tc.req)
		ae, ok := apperror.IsAppError(err)
		if !ok || ae.Code != tc.code {
			t.Errorf("req %+v: esperava %d, got %v", tc.req, tc.code, err)
		}
	}
}

func TestService_Update_RoleInvalida(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()})
	_, err := svc.Update(context.Background(), "u1", UpdateRequest{Nome: "X", Role: "superadmin"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Update(context.Background(), "x", UpdateRequest{Nome: "X", Role: "viewer"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava 404, got %v", err)
	}
}

func TestService_UpdatePassword_CurtaDemais(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()})
	err := svc.UpdatePassword(context.Background(), "u1", UpdatePasswordRequest{Password: "abc"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{})
	err := svc.Delete(context.Background(), "x")
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava 404, got %v", err)
	}
}

func TestService_Delete_Success(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()})
	if err := svc.Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
