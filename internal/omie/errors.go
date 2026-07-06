package omie

import (
	"fmt"
	"strings"
)

// OmieError representa um erro retornado pela API do Omie.
type OmieError struct {
	FaultCode   string `json:"faultcode"`
	FaultString string `json:"faultstring"`
}

func (e OmieError) Error() string {
	return fmt.Sprintf("omie error [%s]: %s", e.FaultCode, e.FaultString)
}

// Códigos de erro conhecidos do Omie
const (
	ErrCodeCredencialInvalida = "SOAP-ENV:Client-107"
	ErrCodeLimiteExcedido     = "SOAP-ENV:Client-108"
	ErrCodeRegistroNaoEnc     = "SOAP-ENV:Client-500"
)

func IsCredencialInvalida(err error) bool {
	if e, ok := err.(OmieError); ok {
		return e.FaultCode == ErrCodeCredencialInvalida
	}
	return false
}

func IsLimiteExcedido(err error) bool {
	if e, ok := err.(OmieError); ok {
		return e.FaultCode == ErrCodeLimiteExcedido
	}
	return false
}

// IsSemRegistros detecta o erro 500 do Omie que indica ausência de registros para a página
// solicitada — ocorre em sincronizações incrementais quando não há dados novos no período.
// Nesse caso o comportamento correto é tratar como 0 registros sincronizados com sucesso.
func IsSemRegistros(err error) bool {
	if e, ok := err.(OmieError); ok {
		return strings.Contains(e.FaultString, "Não existem registros para a página")
	}
	return false
}
