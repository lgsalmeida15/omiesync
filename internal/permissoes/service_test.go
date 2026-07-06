package permissoes

import (
	"context"
	"testing"
	"time"

	"omie-sync-api/internal/apperror"
)

type mockRepo struct {
	permissao  *Permissao
	permissoes []*Permissao
	err        error
}

func (m *mockRepo) Grant(_ context.Context, uID, eID, rec, acao string) (*Permissao, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Permissao{ID: "p1", UsuarioID: uID, EmpresaID: eID, Recurso: rec, Acao: acao, CreatedAt: time.Now()}, nil
}
func (m *mockRepo) Revoke(_ context.Context, _, _, _, _ string) error { return m.err }
func (m *mockRepo) ListByUsuario(_ context.Context, _ string) ([]*Permissao, error) {
	return m.permissoes, m.err
}
func (m *mockRepo) ListByEmpresa(_ context.Context, _ string) ([]*Permissao, error) {
	return m.permissoes, m.err
}
func (m *mockRepo) Has(_ context.Context, _, _, _, _ string) (bool, error) { return true, m.err }

func TestService_Grant_Success(t *testing.T) {
	svc := NewService(&mockRepo{})
	p, err := svc.Grant(context.Background(), GrantRequest{
		UsuarioID: "u1", EmpresaID: "e1", Recurso: "sync", Acao: "ver",
	})
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if p.Recurso != "sync" || p.Acao != "ver" {
		t.Errorf("permissão incorreta: %+v", p)
	}
}

func TestService_Grant_RecursoInvalido(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Grant(context.Background(), GrantRequest{
		UsuarioID: "u1", EmpresaID: "e1", Recurso: "invalido", Acao: "ver",
	})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Grant_AcaoInvalida(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.Grant(context.Background(), GrantRequest{
		UsuarioID: "u1", EmpresaID: "e1", Recurso: "dashboard", Acao: "deletar",
	})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Revoke_Success(t *testing.T) {
	svc := NewService(&mockRepo{})
	err := svc.Revoke(context.Background(), RevokeRequest{
		UsuarioID: "u1", EmpresaID: "e1", Recurso: "admin", Acao: "editar",
	})
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

func TestService_ListByUsuario(t *testing.T) {
	repo := &mockRepo{permissoes: []*Permissao{
		{ID: "p1", Recurso: "sync", Acao: "ver"},
		{ID: "p2", Recurso: "dashboard", Acao: "editar"},
	}}
	svc := NewService(repo)
	ps, err := svc.ListByUsuario(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListByUsuario: %v", err)
	}
	if len(ps) != 2 {
		t.Errorf("len: got %d want 2", len(ps))
	}
}

func TestService_TodosRecursosEAcoes(t *testing.T) {
	svc := NewService(&mockRepo{})
	recursos := []string{"dashboard", "sync", "admin"}
	acoes := []string{"ver", "editar", "forcar_sync"}

	for _, rec := range recursos {
		for _, acao := range acoes {
			_, err := svc.Grant(context.Background(), GrantRequest{
				UsuarioID: "u1", EmpresaID: "e1", Recurso: rec, Acao: acao,
			})
			if err != nil {
				t.Errorf("recurso=%q acao=%q não deveria falhar: %v", rec, acao, err)
			}
		}
	}
}
