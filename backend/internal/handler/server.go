// Package handler は、HTTP の境界の処理を表す。API 契約から生成されたインターフェースを実装し、
// リクエストをユースケースに渡して、結果をレスポンスにする。
package handler

import (
	"context"

	"github.com/kajiya-i/muda/backend/internal/infrastructure/httpapi/apigen"
)

// Server は、API 契約から生成された StrictServerInterface を実装する。
type Server struct{}

var _ apigen.StrictServerInterface = Server{}

// New は、Server を返す。
func New() Server {
	return Server{}
}

// GetHealth は、バックエンドが動いていることを返す。
func (Server) GetHealth(_ context.Context, _ apigen.GetHealthRequestObject) (apigen.GetHealthResponseObject, error) {
	return apigen.GetHealth200Response{}, nil
}
