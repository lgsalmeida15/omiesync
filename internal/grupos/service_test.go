package grupos

import (
	"context"
	"errors"
	"testing"

	"omie-sync-api/internal/apperror"
)

// --- mock repository ---

type mockRepo struct {
	grupo          *Grupo
	grupos         []*Grupo
	total          int64
	insertErr      error
	updateErr      error
	deleteErr      error
	slugExists     bool
	empresasAtivas int64
}

func (m *mockRepo) Insert(_ context.Context, nome, slug, schemaName string) (*Grupo, error) {
	if m.insertErr != nil {
		return nil, m.insertErr
	}
	return &Grupo{ID: "new-id", Nome: nome, Slug: slug, SchemaName: schemaName, Status: "ativo"}, nil
}

func (m *mockRepo) GetByID(_ context.Context, _ string) (*Grupo, error) {
	if m.grupo == nil {
		return nil, errors.New("nÃ£o encontrado")
	}
	return m.grupo, nil
}

func (m *mockRepo) GetBySlug(_ context.Context, _ string) (*Grupo, error) {
	if m.slugExists {
		return &Grupo{ID: "existing"}, nil
	}
	return nil, errors.New("nÃ£o encontrado")
}

func (m *mockRepo) List(_ context.Context, _, _ int32) ([]*Grupo, error) {
	return m.grupos, nil
}

func (m *mockRepo) Count(_ context.Context) (int64, error) {
	return m.total, nil
}

func (m *mockRepo) Update(_ context.Context, id, nome string) (*Grupo, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &Grupo{ID: id, Nome: nome}, nil
}

func (m *mockRepo) SoftDelete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockRepo) CountEmpresasAtivas(_ context.Context, _ string) (int64, error) {
	return m.empresasAtivas, nil
}

// --- testes ---

func TestService_Create_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)

	g, err := svc.Create(context.Background(), CreateRequest{Nome: "Alpha", Slug: "alpha"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.Slug != "alpha" {
		t.Errorf("slug: got %q want %q", g.Slug, "alpha")
	}
	if g.SchemaName != "grupo_alpha" {
		t.Errorf("schema_name: got %q want %q", g.SchemaName, "grupo_alpha")
	}
}

func TestService_Create_SlugUppercaseNormalized(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)

	g, err := svc.Create(context.Background(), CreateRequest{Nome: "Beta", Slug: "BETA"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.Slug != "beta" {
		t.Errorf("slug nÃ£o normalizado: got %q", g.Slug)
	}
}

func TestService_Create_DuplicateSlug(t *testing.T) {
	svc := NewService(&mockRepo{slugExists: true}, nil)

	_, err := svc.Create(context.Background(), CreateRequest{Nome: "Alpha", Slug: "alpha"})
	if err == nil {
		t.Fatal("esperava erro de conflito")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava AppError 409, got %v", err)
	}
}

func TestService_Create_InvalidSlug(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)

	// Uppercase Ã© normalizado para lowercase, entÃ£o Ã© vÃ¡lido.
	// InvÃ¡lidos: espaÃ§os, caracteres especiais (exceto - e _), vazio.
	cases := []string{"alpha beta", "alpha!", "alpha@test", ""}
	for _, slug := range cases {
		_, err := svc.Create(context.Background(), CreateRequest{Nome: "Test", Slug: slug})
		if err == nil {
			t.Errorf("slug %q deveria ser invÃ¡lido", slug)
		}
	}
}

func TestService_Create_MissingNome(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)

	_, err := svc.Create(context.Background(), CreateRequest{Nome: "", Slug: "ok"})
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava AppError 422, got %v", err)
	}
}

func TestService_Delete_WithActiveEmpresas(t *testing.T) {
	svc := NewService(&mockRepo{
		grupo:          &Grupo{ID: "g1"},
		empresasAtivas: 3,
	}, nil)

	err := svc.Delete(context.Background(), "g1")
	if err == nil {
		t.Fatal("esperava erro de conflito")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava AppError 409, got %v", err)
	}
}

func TestService_Delete_NoActiveEmpresas(t *testing.T) {
	svc := NewService(&mockRepo{
		grupo:          &Grupo{ID: "g1"},
		empresasAtivas: 0,
	}, nil)

	err := svc.Delete(context.Background(), "g1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{}, nil) // grupo == nil

	err := svc.Delete(context.Background(), "inexistente")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava AppError 404, got %v", err)
	}
}

func TestService_List_DefaultPagination(t *testing.T) {
	svc := NewService(&mockRepo{
		grupos: []*Grupo{{ID: "g1"}, {ID: "g2"}},
		total:  2,
	}, nil)

	gs, total, err := svc.List(context.Background(), ListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(gs) != 2 {
		t.Errorf("len: got %d want 2", len(gs))
	}
	if total != 2 {
		t.Errorf("total: got %d want 2", total)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{}, nil) // grupo == nil

	_, err := svc.Update(context.Background(), "x", UpdateRequest{Nome: "novo"})
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava AppError 404, got %v", err)
	}
}

