package main

import (
	"errors"
	"fmt"
	"github.com/danyelkeddah/snippetbox/internal/models"
	"github.com/danyelkeddah/snippetbox/internal/validator"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

type SnippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

type UserSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type UserLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (a *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := a.snippets.Latest()
	if err != nil {
		a.serverError(w, err)
		return
	}

	data := a.NewTemplateData(r)
	data.Snippets = snippets
	a.render(w, http.StatusOK, "home", data)
}

func (a *application) snippetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		a.notFound(w)
		return
	}
	snippet, err := a.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			a.notFound(w)
		} else {
			a.serverError(w, err)
		}
		return
	}

	// Get the value and delete it from cache (acts like on time fetch),
	// If there is no matching key in session it will return empty string.

	data := a.NewTemplateData(r)
	data.Snippet = snippet
	a.render(w, http.StatusOK, "view", data)
}

func (a *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := a.NewTemplateData(r)
	data.Form = SnippetCreateForm{
		Expires: 365,
	}
	a.render(w, http.StatusOK, "create", data)
}

func (a *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form SnippetCreateForm
	err := a.decodePostForm(r, &form)
	if err != nil {
		a.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank.")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field can not be more than 100 characters long.")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank.")
	form.CheckField(validator.PermittedValue(form.Expires, 1, 7, 365), "expires", "This field must equal 1, 7 or 365")

	if !form.Valid() {
		data := a.NewTemplateData(r)
		data.Form = form
		a.render(w, http.StatusUnprocessableEntity, "create", data)
		return
	}

	id, err := a.snippets.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		a.serverError(w, err)
		return
	}

	a.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (a *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := a.NewTemplateData(r)
	data.Form = UserSignupForm{}
	a.render(w, http.StatusOK, "signup", data)
}

func (a *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form UserSignupForm
	err := a.decodePostForm(r, &form)
	if err != nil {
		a.clientError(w, http.StatusBadRequest)
		return
	}
	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRx), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	if form.Invalid() {
		data := a.NewTemplateData(r)
		data.Form = form
		a.render(w, http.StatusUnprocessableEntity, "signup", data)
		return
	}

	err = a.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address ia already in use")
			data := a.NewTemplateData(r)
			data.Form = form
			a.render(w, http.StatusUnprocessableEntity, "signup", data)
			return
		} else {
			a.serverError(w, err)
		}
		return
	}
	a.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (a *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := a.NewTemplateData(r)
	data.Form = UserLoginForm{}

	a.render(w, http.StatusOK, "login", data)
}

func (a *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form UserLoginForm
	err := a.decodePostForm(r, &form)

	if err != nil {
		a.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRx), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if form.Invalid() {
		data := a.NewTemplateData(r)
		data.Form = form
		a.render(w, http.StatusUnprocessableEntity, "login", data)

		return
	}
	id, err := a.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := a.NewTemplateData(r)
			data.Form = form
			a.render(w, http.StatusUnprocessableEntity, "login", data)
		} else {
			a.serverError(w, err)
		}
		return
	}
	// regenerate user session
	err = a.sessionManager.RenewToken(r.Context()) // will change the id of the current user session retain the data
	if err != nil {
		a.serverError(w, err)
		return
	}
	// set id in user session
	a.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (a *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := a.sessionManager.RenewToken(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.sessionManager.Remove(r.Context(), "authenticatedUserID")
	a.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
