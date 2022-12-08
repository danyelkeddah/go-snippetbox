package main

import (
	"github.com/danyelkeddah/snippetbox/ui"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"net/http"
)

func (a *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.notFound(w)
	})

	fileServer := http.FileServer(http.FS(ui.Files))

	router.Handler(http.MethodGet, "/static/*filepath", fileServer)
	dynamic := alice.New(a.sessionManager.LoadAndSave, noSurf, a.authenticate)
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(a.home))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(a.snippetView))
	protected := dynamic.Append(a.requiredAuthentication)
	router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(a.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(a.snippetCreatePost))

	// Authentication routes
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(a.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(a.userSignupPost))

	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(a.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(a.userLoginPost))

	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(a.userLogoutPost))

	standard := alice.New(a.recoverPanic, a.logRequest, secureHeaders)
	return standard.Then(router)
}
