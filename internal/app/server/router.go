package server

import (
	"github.com/PiskarevSA/go-advanced/internal/handlers"
	"github.com/go-chi/chi/v5"
)

// TODO PR #5
// классно было бы сделать промежуточный слой.
// Структура handlers, внутри которая будет структура usecase/service (как
// угодно назови) -- структура отвечающая за бизнес логику, и внутри этой
// структуры уже всё необходимое для исполнения твоей бизнес логики.
//
// Просто такой подход не очень гибкий, потому что помимо репозитория может
// понадобиться выполнить какую-то бизнес логику, сделать какую-нибудь отправку
// события в кафку итд итп.
// Хэндлеру обо всей этой логике знать совршенно не обязательно. Неплохо, когда
// он работает по принципу черной коробки
//
// (Почитай про чистую архитектуру)

func MetricsRouter(repo handlers.Repositories) chi.Router {
	r := chi.NewRouter()
	r.Get(`/`, handlers.MainPage(repo))
	r.Post(`/update/{type}/{name}/{value}`, handlers.Update(repo))
	r.Get(`/value/{type}/{name}`, handlers.Get(repo))
	return r
}
